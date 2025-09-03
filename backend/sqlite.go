package backend

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteVec struct {
	db *sql.DB
}

func NewSQLiteVec(ctx context.Context, dbPath string) (*SQLiteVec, error) {
	sqlite_vec.Auto()

	dsn := fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}

	return &SQLiteVec{
		db: db,
	}, nil
}

func (s *SQLiteVec) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("closing sqlite database: %w", err)
	}

	return nil
}

// AddDefinition adds a new definition and associated features to the database.
//
// * first we look up if the word already exists,
// * otherwise the word is added to the words table;
// * then its features are added to the features table;
// * then the embeddings for each feature are added to the embeddings table.
func (s *SQLiteVec) AddDefinition(
	ctx context.Context,
	definition Definition,
) (_ int64, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var wordID int64

	if s.db.QueryRowContext(
		ctx,
		`
			SELECT id
			FROM words
			WHERE word = ? AND definition = ?
		`,
		definition.Word,
		definition.Definition,
	).Scan(&wordID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("querying word: %w", err)
	}

	if wordID == 0 {
		if err := s.db.QueryRowContext(
			ctx,
			`
				INSERT INTO words (word, definition, example, author)
				VALUES (?, ?, ?, ?)
				RETURNING id
			`,
			definition.Word,
			definition.Definition,
			definition.Example,
			definition.Author,
		).Scan(&wordID); err != nil {
			return 0, fmt.Errorf("inserting new word details: %w", err)
		}
	}

	s.AddFeatures(ctx, wordID, definition.Features)

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing transaction: %w", err)
	}

	return wordID, nil
}

func (s *SQLiteVec) RelatedWords(
	ctx context.Context,
	model Model,
	vector Embedding,
	limit int,
) ([]SimilarDefinition, error) {
	vec, err := sqlite_vec.SerializeFloat32(vector)
	if err != nil {
		return nil, fmt.Errorf("serializing embedding: %w", err)
	}

	stmt, err := s.db.PrepareContext(
		ctx,
		`
		SELECT w.word, w.definition, w.example, w.author, best.distance, wf.phrase
		FROM words w
		JOIN (
			SELECT
				wf.id,
				wf.word_id,
				vec_distance_cosine(e.embedding, ?) AS distance
			FROM word_features wf
			JOIN embeddings e ON e.word_feature_id = wf.id
			WHERE (wf.word_id, distance) IN (
				SELECT
					wf2.word_id,
					MIN(vec_distance_cosine(e2.embedding, ?))
				FROM word_features wf2
				JOIN embeddings e2 ON e2.word_feature_id = wf2.id
				GROUP BY wf2.word_id
			) AND e.embedding_model_id = ?
		) best ON w.id = best.word_id
		JOIN word_features wf ON wf.id = best.id
		ORDER BY best.distance ASC
		LIMIT ?
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("preparing statement: %w", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, vec, vec, model, limit)
	if err != nil {
		return nil, fmt.Errorf("querying statement: %w", err)
	}

	var definitions []SimilarDefinition

	for rows.Next() {
		var definition SimilarDefinition
		if err := rows.Scan(
			&definition.Word.Word,
			&definition.Word.Definition,
			&definition.Word.Example,
			&definition.Word.Author,
			&definition.Distance,
			&definition.Phrase,
		); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		definitions = append(definitions, definition)
	}

	return definitions, nil
}

func (s *SQLiteVec) AddFeatures(
	ctx context.Context,
	wordID int64,
	features []Feature,
) error {
	featureIDs := make([]int64, 0, len(features))

	for _, feature := range features {
		id, err := s.addFeature(ctx, wordID, feature)
		if err != nil {
			return fmt.Errorf("adding feature: %w", err)
		}

		featureIDs = append(featureIDs, id)
	}

	embeddingInsertionStatement, err := s.db.PrepareContext(
		ctx,
		`
			INSERT INTO embeddings (
				word_feature_id,
				embedding_model_id,
				embedding
			) VALUES (?, ?, ?)
			ON CONFLICT(word_feature_id, embedding_model_id) DO UPDATE SET
				embedding = excluded.embedding
		`,
	)
	if err != nil {
		return fmt.Errorf("preparing query statement: %w", err)
	}

	defer embeddingInsertionStatement.Close()

	for i, feature := range features {
		for model, embedding := range feature.Embeddings {
			embeddingBytes, err := sqlite_vec.SerializeFloat32(embedding)
			if err != nil {
				return fmt.Errorf("serializing embedding: %w", err)
			}

			if _, err := embeddingInsertionStatement.ExecContext(
				ctx,
				featureIDs[i],
				model,
				embeddingBytes,
			); err != nil {
				return fmt.Errorf("inserting embedding: %w", err)
			}
		}
	}

	return nil
}

func (s *SQLiteVec) addFeature(
	ctx context.Context,
	wordID int64,
	feature Feature,
) (int64, error) {
	var id int64

	if err := s.db.QueryRowContext(
		ctx,
		`
			SELECT id
			FROM word_features
			WHERE word_id = ? AND phrase = ?
		`,
		wordID,
		feature.Phrase,
	).Scan(&id); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("querying existing feature: %w", err)
	} else if err == nil {
		// If we found the existing feature ID, return it.
		return id, nil
	}

	if err := s.db.QueryRowContext(
		ctx,
		`
		INSERT INTO word_features (word_id, phrase, autogenerated)
		VALUES (?, ?, ?)
		RETURNING id
		`,
		wordID,
		feature.Phrase,
		feature.Autogenerated,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("inserting new feature: %w", err)
	}

	return id, nil
}

// GetWordFeatures returns all the features of a specific word.
func (s *SQLiteVec) GetWordFeatures(
	ctx context.Context,
	wordID int64,
) ([]Feature, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`
			SELECT id, phrase, autogenerated
			FROM word_features
			WHERE word_id = ?
		`,
		wordID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying features: %w", err)
	}

	embeddingsQuery, err := s.db.PrepareContext(
		ctx,
		`
			SELECT embedding_model_id, vec_to_json(embedding)
			FROM embeddings
			WHERE word_feature_id = ?
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("preparing embeddings query: %w", err)
	}

	var features []Feature

	for rows.Next() {
		var (
			featureID int64
			feature   Feature
		)

		if err := rows.Scan(
			&featureID,
			&feature.Phrase,
			&feature.Autogenerated,
		); err != nil {
			return nil, fmt.Errorf("scanning feature row: %w", err)
		}

		embeddingRows, err := embeddingsQuery.QueryContext(ctx, featureID)
		if err != nil {
			return nil, fmt.Errorf("querying feature embeddings: %w", err)
		}

		feature.Embeddings = make(map[Model]Embedding)

		for embeddingRows.Next() {
			var (
				model     Model
				embedding []float32
			)

			if err := embeddingRows.Scan(
				&model,
				jsonValue(&embedding),
			); err != nil {
				return nil, fmt.Errorf("scanning embedding row: %w", err)
			}

			feature.Embeddings[model] = embedding
		}

		features = append(features, feature)
	}

	return features, nil
}

func (s *SQLiteVec) GetRandomDefinition(
	ctx context.Context,
) (*Word, error) {
	stmt, err := s.db.PrepareContext(
		ctx,
		`
		SELECT word, definition, example, author
		FROM words
		ORDER BY RANDOM()
		LIMIT 1
		`,
	)
	if err != nil {
		return nil, fmt.Errorf("preparing statement: %w", err)
	}

	defer stmt.Close()

	var definition Word

	if err := stmt.QueryRowContext(ctx).Scan(
		&definition.Word,
		&definition.Definition,
		&definition.Example,
		&definition.Author,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No definitions found
		}
		return nil, fmt.Errorf("querying random definition: %w", err)
	}

	return &definition, nil
}

// DBWord encapsulates the full [Word] data, and the DB ID that has been
// assigned.
type DBWord struct {
	ID int64
	Word
}

// GetWords retrieves an iterator over all words in the database.
func (s *SQLiteVec) GetWords(
	ctx context.Context,
) iter.Seq2[*DBWord, error] {
	return func(yield func(*DBWord, error) bool) {
		stmt, err := s.db.PrepareContext(
			ctx,
			`
		SELECT id, word, definition, example, author
		FROM words
		ORDER BY id
		`,
		)
		if err != nil {
			yield(nil, fmt.Errorf("preparing statement: %w", err))

			return
		}

		defer stmt.Close()

		rows, err := stmt.QueryContext(ctx)
		if err != nil {
			yield(nil, fmt.Errorf("querying statement: %w", err))

			return
		}

		defer rows.Close()

		for rows.Next() {
			var definition DBWord

			if err := rows.Scan(
				&definition.ID,
				&definition.Word.Word,
				&definition.Word.Definition,
				&definition.Word.Example,
				&definition.Word.Author,
			); err != nil {
				yield(nil, fmt.Errorf("scanning row: %w", err))

				return
			}

			if !yield(&definition, nil) {
				return
			}
		}
	}
}

func (s *SQLiteVec) CompareEmbeddings(
	ctx context.Context,
	embedding1 Embedding,
	embedding2 Embedding,
) (float64, error) {
	vec1, err := sqlite_vec.SerializeFloat32(embedding1)
	if err != nil {
		return 0, fmt.Errorf("serializing first embedding: %w", err)
	}

	vec2, err := sqlite_vec.SerializeFloat32(embedding2)
	if err != nil {
		return 0, fmt.Errorf("serializing second embedding: %w", err)
	}

	stmt, err := s.db.PrepareContext(
		ctx,
		`
		SELECT vec_distance_cosine(?, ?)
		`,
	)
	if err != nil {
		return 0, fmt.Errorf("preparing statement: %w", err)
	}

	defer stmt.Close()

	var distance float64
	if err := stmt.QueryRowContext(ctx, vec1, vec2).Scan(&distance); err != nil {
		return 0, fmt.Errorf("querying distance: %w", err)
	}

	return distance, nil
}

// jsonValue is a helper function to create a JSONScanner for a given value.
func jsonValue[T any](v *T) JSONScanner[T] {
	return JSONScanner[T]{
		Value: v,
	}
}

// JSONScanner is a wrapper that allows a JSON-encoded value to be scanned from
// the DB.
type JSONScanner[T any] struct {
	Value *T
}

// Scan retrieves the JSON-encoded value from the DB and unmarshals it into the
// provided pointer.
func (j JSONScanner[T]) Scan(src any) error {
	var data []byte

	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("expected []byte or string, got %T", src)
	}

	if err := json.Unmarshal(data, j.Value); err != nil {
		return fmt.Errorf("unmarshalling DB JSON to %T: %w", j, err)
	}

	return nil
}

// Ensure [JSONScanner] implements the [sql.Scanner] interface.
var _ sql.Scanner = &JSONScanner[any]{}

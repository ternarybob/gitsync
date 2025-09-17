package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

type Store struct {
	db         *bbolt.DB
	bucketName []byte
}

type Transaction struct {
	ID         string    `json:"id"`
	JobName    string    `json:"job_name"`
	RepoName   string    `json:"repo_name"`
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	Status     string    `json:"status"`
	CommitHash string    `json:"commit_hash"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Error      string    `json:"error,omitempty"`
}

const (
	StatusPending = "pending"
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

func New(path string, bucketName string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := createDirIfNotExists(dir); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	bucket := []byte(bucketName)

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Store{
		db:         db,
		bucketName: bucket,
	}, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) SaveTransaction(tx *Transaction) error {
	if tx.ID == "" {
		tx.ID = generateID()
	}

	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	return s.db.Update(func(btx *bbolt.Tx) error {
		b := btx.Bucket(s.bucketName)
		return b.Put([]byte(tx.ID), data)
	})
}

func (s *Store) GetTransaction(id string) (*Transaction, error) {
	var tx Transaction

	err := s.db.View(func(btx *bbolt.Tx) error {
		b := btx.Bucket(s.bucketName)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("transaction not found: %s", id)
		}
		return json.Unmarshal(data, &tx)
	})

	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (s *Store) GetTransactionsByJob(jobName string, limit int) ([]*Transaction, error) {
	var transactions []*Transaction

	err := s.db.View(func(btx *bbolt.Tx) error {
		b := btx.Bucket(s.bucketName)
		c := b.Cursor()

		count := 0
		for k, v := c.Last(); k != nil && count < limit; k, v = c.Prev() {
			var tx Transaction
			if err := json.Unmarshal(v, &tx); err != nil {
				continue
			}
			if tx.JobName == jobName {
				transactions = append(transactions, &tx)
				count++
			}
		}
		return nil
	})

	return transactions, err
}

func (s *Store) GetLastSuccessfulSync(jobName, repoName, target string) (*Transaction, error) {
	var lastTx *Transaction

	err := s.db.View(func(btx *bbolt.Tx) error {
		b := btx.Bucket(s.bucketName)
		c := b.Cursor()

		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			var tx Transaction
			if err := json.Unmarshal(v, &tx); err != nil {
				continue
			}
			if tx.JobName == jobName && tx.RepoName == repoName && tx.Target == target && tx.Status == StatusSuccess {
				lastTx = &tx
				return nil
			}
		}
		return nil
	})

	return lastTx, err
}

func (s *Store) CleanupOldTransactions(retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	return s.db.Update(func(btx *bbolt.Tx) error {
		b := btx.Bucket(s.bucketName)
		c := b.Cursor()

		var toDelete [][]byte
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var tx Transaction
			if err := json.Unmarshal(v, &tx); err != nil {
				continue
			}
			if tx.EndTime.Before(cutoff) {
				toDelete = append(toDelete, k)
			}
		}

		for _, key := range toDelete {
			if err := b.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	err := s.db.View(func(btx *bbolt.Tx) error {
		b := btx.Bucket(s.bucketName)

		totalCount := 0
		successCount := 0
		failedCount := 0
		var totalDuration time.Duration

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var tx Transaction
			if err := json.Unmarshal(v, &tx); err != nil {
				continue
			}

			totalCount++
			if tx.Status == StatusSuccess {
				successCount++
				if !tx.EndTime.IsZero() && !tx.StartTime.IsZero() {
					totalDuration += tx.EndTime.Sub(tx.StartTime)
				}
			} else if tx.Status == StatusFailed {
				failedCount++
			}
		}

		stats["total_transactions"] = totalCount
		stats["successful_syncs"] = successCount
		stats["failed_syncs"] = failedCount
		if successCount > 0 {
			stats["avg_duration_seconds"] = totalDuration.Seconds() / float64(successCount)
		} else {
			stats["avg_duration_seconds"] = 0
		}

		return nil
	})

	return stats, err
}

func createDirIfNotExists(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func generateID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), generateRandomString(8))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

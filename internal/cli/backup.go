package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"inventar/internal/store"
)

// backup writes a snapshot of the data directory to a folder laid out like a data directory:
// the database via VACUUM INTO (safe while the server runs) and the photos. Only new or changed
// photos are copied, so nightly runs stay cheap; deleted photos stay in the backup.
// Certificates are not backed up, they can be obtained again.
func backup(ctx context.Context, c command) error {
	var to string
	err := c.parseFlags("backup", func(f *flag.FlagSet) {
		f.StringVar(&to, "to", "", "target folder of the backup")
	}, &to)
	if err != nil {
		return err
	}
	if same, err := samePath(to, c.cfg.DataDir); err != nil {
		return err
	} else if same {
		return errors.New("the backup folder must not be the data directory")
	}
	if err := os.MkdirAll(filepath.Join(to, store.PhotoDir), 0o750); err != nil {
		return fmt.Errorf("create backup folder: %w", err)
	}

	st, err := store.Open(ctx, c.cfg.DataDir)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := snapshotDB(ctx, st, filepath.Join(to, store.DBFile)); err != nil {
		return err
	}

	copied, unchanged, err := syncPhotos(filepath.Join(c.cfg.DataDir, store.PhotoDir), filepath.Join(to, store.PhotoDir))
	if err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "Backup written to %s: database snapshot, %s copied, %d unchanged.\n",
		to, plural(copied, "photo", "photos"), unchanged)
	return nil
}

// snapshotDB writes the snapshot next to the target first and only then replaces the previous
// one, so a failed run never leaves a broken database in the backup.
func snapshotDB(ctx context.Context, st *store.Store, path string) error {
	tmp := path + ".tmp"
	if err := os.Remove(tmp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := st.Snapshot(ctx, tmp); err != nil {
		return fmt.Errorf("database snapshot: %w", err)
	}
	return os.Rename(tmp, path)
}

// syncPhotos copies every photo whose size or modification time differs from the copy in to.
func syncPhotos(from, to string) (copied, unchanged int, err error) {
	err = filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if old, err := os.Stat(target); err == nil && old.Size() == info.Size() && old.ModTime().Equal(info.ModTime()) {
			unchanged++
			return nil
		}
		if err := copyFile(path, target, info); err != nil {
			return fmt.Errorf("copy photo %s: %w", rel, err)
		}
		copied++
		return nil
	})
	return copied, unchanged, err
}

// copyFile copies via a temporary file and takes over the modification time, by which the next
// run recognises the file as unchanged.
func copyFile(from, to string, info fs.FileInfo) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp := to + ".tmp"
	dst, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmp)
		return err
	}
	if err := dst.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chtimes(tmp, info.ModTime(), info.ModTime()); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, to)
}

func samePath(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, err
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, err
	}
	return absA == absB, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

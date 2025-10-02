package testhelpers

import (
	"database/sql"
	"io/ioutil"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB, dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		content, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return err
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return err
		}
	}

	return nil
}

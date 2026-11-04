package example

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/adejoux/tfrefdoc/internal/tfschema"
)

// Path returns the conventional example file of an entity, relative to the
// examples directory, as expected by tfplugindocs.
func Path(e tfschema.Entity) string {
	if e.Kind == tfschema.KindProvider {
		return filepath.Join("provider", "provider.tf")
	}
	return filepath.Join(e.Kind.Dir(), e.Name, e.Kind.ExampleFile())
}

// ImportPath returns the conventional import example of a resource.
func ImportPath(name string) string {
	return filepath.Join("resources", name, "import.sh")
}

// File is an example file to create.
type File struct {
	Path    string // relative to the examples directory
	Content string
}

// Missing returns the example files that do not exist yet in dir. Existing
// files are never overwritten. Deprecated entities are skipped.
func (g *Generator) Missing(dir string, withImport bool) ([]File, error) {
	var out []File
	for _, e := range g.p.Entities() {
		if e.Block.Deprecated {
			continue
		}
		rel := Path(e)
		if ok, err := exists(filepath.Join(dir, rel)); err != nil {
			return nil, err
		} else if !ok {
			out = append(out, File{Path: rel, Content: g.Example(e)})
		}
		if withImport && e.Kind == tfschema.KindResource {
			rel := ImportPath(e.Name)
			if ok, err := exists(filepath.Join(dir, rel)); err != nil {
				return nil, err
			} else if !ok {
				out = append(out, File{Path: rel, Content: ImportExample(e.Name)})
			}
		}
	}
	return out, nil
}

// Write creates the files under dir.
func Write(dir string, files []File) error {
	for _, f := range files {
		path := filepath.Join(dir, f.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Unreviewed returns the files under dir that still contain the TODO marker
// of a generated skeleton.
func Unreviewed(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == dir {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".tf" && ext != ".sh" && ext != ".hcl" {
			return nil
		}
		if ok, err := containsMarker(path); err != nil {
			return err
		} else if ok {
			rel, _ := filepath.Rel(dir, path)
			out = append(out, rel)
		}
		return nil
	})
	return out, err
}

func containsMarker(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.Contains(sc.Text(), TodoMarker) {
			return true, nil
		}
	}
	return false, sc.Err()
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

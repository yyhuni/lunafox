package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type LocalWorkspace struct {
	root     string
	realRoot string
}

func NewLocalWorkspace(root string) (*LocalWorkspace, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." || root == string(filepath.Separator) {
		return nil, fmt.Errorf("%w: workspace root is unsafe", ErrInvalidArgument)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("%w: workspace root is unsafe", ErrInvalidArgument)
	}
	root = filepath.Clean(absoluteRoot)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("%w: workspace root must be a real directory", ErrInvalidArgument)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	return &LocalWorkspace{root: root, realRoot: filepath.Clean(realRoot)}, nil
}

func (workspace *LocalWorkspace) Create(ctx context.Context, _ uuid.UUID) (string, error) {
	if workspace == nil || workspace.root == "" {
		return "", fmt.Errorf("workspace is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := workspace.validateRoot(); err != nil {
		return "", err
	}
	for attempt := 0; attempt < 3; attempt++ {
		key := uuid.NewString()
		path := filepath.Join(workspace.root, key)
		if err := os.Mkdir(path, 0o700); err == nil {
			return path, nil
		} else if !os.IsExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("could not allocate isolated workspace")
}

func (workspace *LocalWorkspace) Remove(ctx context.Context, path string) error {
	if workspace == nil || workspace.root == "" {
		return fmt.Errorf("workspace is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := workspace.validateRoot(); err != nil {
		return err
	}
	if !workspace.isDirectChild(path) {
		return fmt.Errorf("workspace path escaped root")
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("workspace residual is not a directory")
	}
	resolvedRoot := workspace.realRoot
	if resolvedRoot == "" {
		var err error
		resolvedRoot, err = filepath.EvalSymlinks(workspace.root)
		if err != nil {
			return err
		}
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	if !isWithin(resolvedRoot, resolvedPath) || resolvedRoot == resolvedPath {
		return fmt.Errorf("workspace real path escaped root")
	}
	return os.RemoveAll(resolvedPath)
}

func (workspace *LocalWorkspace) RemoveResiduals(ctx context.Context) (int, error) {
	return workspace.RemoveResidualsExcept(ctx, nil)
}

func (workspace *LocalWorkspace) RemoveResidualsExcept(ctx context.Context, protected map[string]struct{}) (int, error) {
	if workspace == nil || workspace.root == "" {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := workspace.validateRoot(); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(workspace.root)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if _, err := uuid.Parse(entry.Name()); err != nil {
			continue
		}
		if _, keep := protected[entry.Name()]; keep {
			continue
		}
		if err := workspace.Remove(ctx, filepath.Join(workspace.root, entry.Name())); err != nil {
			continue
		}
		removed++
	}
	return removed, nil
}

func (workspace *LocalWorkspace) isDirectChild(path string) bool {
	path = filepath.Clean(path)
	parent := filepath.Dir(path)
	return parent == workspace.root && path != workspace.root
}

// validateRoot closes the replacement window between workspace construction
// and a later task/retention operation. A changed root must fail closed rather
// than allowing a task directory to be created through an attacker-controlled
// symlink or a directory with broader permissions.
func (workspace *LocalWorkspace) validateRoot() error {
	if workspace == nil || workspace.root == "" || workspace.realRoot == "" {
		return fmt.Errorf("workspace is not configured")
	}
	info, err := os.Lstat(workspace.root)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("workspace root is not a private directory")
	}
	realRoot, err := filepath.EvalSymlinks(workspace.root)
	if err != nil || filepath.Clean(realRoot) != workspace.realRoot {
		return fmt.Errorf("workspace root changed")
	}
	return nil
}

func isWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

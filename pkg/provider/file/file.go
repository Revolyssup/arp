package file

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Revolyssup/arp/pkg/config"
	"github.com/Revolyssup/arp/pkg/logger"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type FileProvider struct {
	config   config.ProviderConfig
	logger   *logger.Logger
	filePath string
	lastHash string
}

func NewFileProvider(cfg config.ProviderConfig, logger *logger.Logger) (*FileProvider, error) {
	filePath, ok := cfg.Config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'path' configuration")
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &FileProvider{
		config:   cfg,
		filePath: absPath,
		logger:   logger.WithComponent("file_provider"),
	}, nil
}

func (fp *FileProvider) Provide(ch chan<- config.Dynamic) {
	// Create new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fp.logger.Errorf("Failed to create file watcher: %v", err)
		return
	}
	defer watcher.Close()

	err = watcher.Add(fp.filePath)
	if err != nil {
		fp.logger.Errorf("Failed to watch file: %v", err)
		return
	}

	// Also watch the directory to handle file renames/recreations
	dir := filepath.Dir(fp.filePath)
	err = watcher.Add(dir)
	if err != nil {
		fp.logger.Errorf("Failed to watch directory: %v", err)
	}

	// Initial read
	if err := fp.readAndSendConfig(ch); err != nil {
		fp.logger.Errorf("Failed to read initial config: %v", err)
	}

	fp.logger.Debugf("file provider watching: %s", fp.filePath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Handle file events
			if event.Name == fp.filePath {
				if event.Op&fsnotify.Write == fsnotify.Write {
					// File was modified
					if err := fp.readAndSendConfig(ch); err != nil {
						fp.logger.Errorf("Failed to read config: %v", err)
					}
				} else if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
					// File was removed or renamed - try to re-add watch when it reappears
					watcher.Remove(fp.filePath)
				}
			} else if event.Op&fsnotify.Create == fsnotify.Create && event.Name == fp.filePath {
				// File was created (after being removed)
				watcher.Add(fp.filePath)
				if err := fp.readAndSendConfig(ch); err != nil {
					fp.logger.Errorf("Failed to read config: %v", err)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fp.logger.Errorf("File watcher error: %v", err)
		}
	}
}

func (fp *FileProvider) readAndSendConfig(ch chan<- config.Dynamic) error {
	if _, err := os.Stat(fp.filePath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", fp.filePath)
	}

	content, err := os.ReadFile(fp.filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	contentHash := fmt.Sprintf("%x", md5.Sum(content))
	if contentHash == fp.lastHash {
		// Content hasn't changed
		return nil
	}

	var dynamicConfig config.Dynamic
	if err := yaml.Unmarshal(content, &dynamicConfig); err != nil {
		return fmt.Errorf("failed to parse config YAML: %v", err)
	}

	select {
	case ch <- dynamicConfig:
	default:
		fp.logger.Warnf("Warning: Config channel is full, dropping update")
	}

	fp.lastHash = contentHash
	fp.logger.Warnf("file provider sent updated configuration")
	return nil
}

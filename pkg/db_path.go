package pkg

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetPlatformSpecificDBPath() (string, error) {
	if p := os.Getenv("CLOCKIT_TEST_DB"); p != "" {
		return p, nil
	}

	var base string

	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LocalAppData")
		if base == "" {
			base = os.Getenv("AppData")
		}
		if base == "" {
			// fallback to home if envs missing
			h, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(h, "AppData", "Local")
		}
		base = filepath.Join(base, "clockit")
	case "darwin":
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(h, "Library", "Application Support", "clockit")
	default:
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			h, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(h, ".local", "state")
		}
		base = filepath.Join(base, "clockit")
	}

	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}

	return filepath.Join(base, "clockit.db"), nil
}

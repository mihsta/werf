//go:build !linux
// +build !linux

package types

// This is from "github.com/containers/storage".
// StoreOptions is used for passing initialization options to GetStore(), for
// initializing a Store object and the underlying storage that it controls.
type StoreOptions struct {
	RunRoot             string            `json:"runroot,omitempty"`
	GraphRoot           string            `json:"root,omitempty"`
	RootlessStoragePath string            `toml:"rootless_storage_path"`
	GraphDriverName     string            `json:"driver,omitempty"`
	GraphDriverOptions  []string          `json:"driver-options,omitempty"`
	UIDMap              []idtools.IDMap   `json:"uidmap,omitempty"`
	GIDMap              []idtools.IDMap   `json:"gidmap,omitempty"`
	RootAutoNsUser      string            `json:"root_auto_ns_user,omitempty"`
	AutoNsMinSize       uint32            `json:"auto_userns_min_size,omitempty"`
	AutoNsMaxSize       uint32            `json:"auto_userns_max_size,omitempty"`
	PullOptions         map[string]string `toml:"pull_options"`
	DisableVolatile     bool              `json:"disable-volatile,omitempty"`
}

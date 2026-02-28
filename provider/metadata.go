package provider

import (
	"os"
	"syscall"
)

// UnixFileInfo extends FileInfo with Unix-specific metadata
type UnixFileInfo interface {
	FileInfo
	UID() uint32
	GID() uint32
	Mode() os.FileMode
}

// unixFileInfo wraps FileInfo to provide Unix-specific metadata
type unixFileInfo struct {
	FileInfo
	uid  uint32
	gid  uint32
	mode os.FileMode
}

func (u *unixFileInfo) UID() uint32  { return u.uid }
func (u *unixFileInfo) GID() uint32  { return u.gid }
func (u *unixFileInfo) Mode() os.FileMode { return u.mode }

// WrapOSFileInfo converts an os.FileInfo into a UnixFileInfo
func WrapOSFileInfo(info os.FileInfo) UnixFileInfo {
	baseInfo := &localFileInfo{
		name:    info.Name(),
		size:    info.Size(),
		isDir:   info.IsDir(),
		modTime: info.ModTime(),
	}

	sysStat := info.Sys()
	if sysStat == nil {
		return baseInfo
	}
	
	fileStat, ok := sysStat.(*syscall.Stat_t)
	if !ok {
		return baseInfo
	}
	
	return &unixFileInfo{
		FileInfo: baseInfo,
		uid:      fileStat.Uid,
		gid:      fileStat.Gid,
		mode:     info.Mode().Perm(),
	}
}

// NewUnixFileInfo creates a UnixFileInfo from raw values
func NewUnixFileInfo(info FileInfo, uid, gid uint32, mode os.FileMode) UnixFileInfo {
	return &unixFileInfo{
		FileInfo: info,
		uid:      uid,
		gid:      gid,
		mode:     mode,
	}
}

// UIDMapping maps source UIDs to destination UIDs
type UIDMapping map[uint32]uint32

// GIDMapping maps source GIDs to destination GIDs
type GIDMapping map[uint32]uint32

// MetadataMapper handles translation of file metadata between source and destination
type MetadataMapper struct {
	uidMapping UIDMapping
	gidMapping GIDMapping
	// If true, preserve source UID/GID when no mapping exists
	// If false, use destination default (typically the running user)
	preserveUnmapped bool
}

// MetadataMapperOption configures a MetadataMapper
type MetadataMapperOption func(*MetadataMapper)

// WithUIDMapping sets the UID mapping table
func WithUIDMapping(mapping UIDMapping) MetadataMapperOption {
	return func(m *MetadataMapper) {
		m.uidMapping = mapping
	}
}

// WithGIDMapping sets the GID mapping table
func WithGIDMapping(mapping GIDMapping) MetadataMapperOption {
	return func(m *MetadataMapper) {
		m.gidMapping = mapping
	}
}

// WithPreserveUnmapped controls whether unmapped UIDs/GIDs are preserved
func WithPreserveUnmapped(preserve bool) MetadataMapperOption {
	return func(m *MetadataMapper) {
		m.preserveUnmapped = preserve
	}
}

// NewMetadataMapper creates a new MetadataMapper with the given options
func NewMetadataMapper(opts ...MetadataMapperOption) *MetadataMapper {
	m := &MetadataMapper{
		uidMapping:       make(UIDMapping),
		gidMapping:       make(GIDMapping),
		preserveUnmapped: true,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// MapUID returns the destination UID for a source UID
func (m *MetadataMapper) MapUID(uid uint32) (uint32, bool) {
	if mapped, ok := m.uidMapping[uid]; ok {
		return mapped, true
	}
	if m.preserveUnmapped {
		return uid, true
	}
	return 0, false
}

// MapGID returns the destination GID for a source GID
func (m *MetadataMapper) MapGID(gid uint32) (uint32, bool) {
	if mapped, ok := m.gidMapping[gid]; ok {
		return mapped, true
	}
	if m.preserveUnmapped {
		return gid, true
	}
	return 0, false
}

// ApplyMetadata applies file metadata (permissions, ownership) to a file
func ApplyMetadata(path string, fileInfo FileInfo, mapper *MetadataMapper) error {
	unixInfo, ok := fileInfo.(UnixFileInfo)
	if !ok {
		// No Unix metadata to apply
		return nil
	}

	// Apply permissions
	if unixInfo.Mode() != 0 {
		if err := os.Chmod(path, unixInfo.Mode()); err != nil {
			return err
		}
	}

	// Apply ownership if mapper is provided
	if mapper != nil {
		uid, uidOK := mapper.MapUID(unixInfo.UID())
		gid, gidOK := mapper.MapGID(unixInfo.GID())
		if uidOK && gidOK {
			if err := os.Chown(path, int(uid), int(gid)); err != nil {
				return err
			}
		}
	}

	return nil
}

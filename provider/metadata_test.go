package provider

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type dummyUnixFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (d *dummyUnixFileInfo) Name() string       { return d.name }
func (d *dummyUnixFileInfo) Size() int64        { return d.size }
func (d *dummyUnixFileInfo) IsDir() bool        { return d.isDir }
func (d *dummyUnixFileInfo) ModTime() time.Time { return d.modTime }

func TestWrapOSFileInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gofast-meta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(filePath, []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	unixInfo := WrapOSFileInfo(stat)
	if unixInfo.Name() != "test.txt" {
		t.Errorf("expected name 'test.txt', got %s", unixInfo.Name())
	}
	if unixInfo.Size() != 5 {
		t.Errorf("expected size 5, got %d", unixInfo.Size())
	}

	// This is partly OS-dependent, but mode must at least be 0644
	mode := unixInfo.Mode() & os.ModePerm
	if mode != 0644 {
		t.Errorf("expected mode 0644, got %v", mode)
	}
}

func TestMetadataMapper(t *testing.T) {
	uidMap := UIDMapping{
		1000: 2000,
		1001: 2001,
	}
	gidMap := GIDMapping{
		100: 200,
	}

	tests := []struct {
		name     string
		mapper   *MetadataMapper
		uidIn    uint32
		uidOut   uint32
		uidOk    bool
		gidIn    uint32
		gidOut   uint32
		gidOk    bool
	}{
		{
			name:   "mapped values",
			mapper: NewMetadataMapper(WithUIDMapping(uidMap), WithGIDMapping(gidMap)),
			uidIn:  1000, uidOut: 2000, uidOk: true,
			gidIn:  100, gidOut: 200, gidOk: true,
		},
		{
			name:   "unmapped values, preserve mapped",
			mapper: NewMetadataMapper(WithUIDMapping(uidMap), WithGIDMapping(gidMap), WithPreserveUnmapped(true)),
			uidIn:  1002, uidOut: 1002, uidOk: true,
			gidIn:  102, gidOut: 102, gidOk: true,
		},
		{
			name:   "unmapped values, dont preserve",
			mapper: NewMetadataMapper(WithUIDMapping(uidMap), WithGIDMapping(gidMap), WithPreserveUnmapped(false)),
			uidIn:  1002, uidOut: 0, uidOk: false,
			gidIn:  102, gidOut: 0, gidOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid, ok := tt.mapper.MapUID(tt.uidIn)
			if uid != tt.uidOut || ok != tt.uidOk {
				t.Errorf("UID mapping failed. Expected (%d, %v), got (%d, %v)", tt.uidOut, tt.uidOk, uid, ok)
			}

			gid, ok := tt.mapper.MapGID(tt.gidIn)
			if gid != tt.gidOut || ok != tt.gidOk {
				t.Errorf("GID mapping failed. Expected (%d, %v), got (%d, %v)", tt.gidOut, tt.gidOk, gid, ok)
			}
		})
	}
}

func TestUnixFileInfo_Wrapper(t *testing.T) {
	d := &dummyUnixFileInfo{name: "fake"}
	ui := NewUnixFileInfo(d, 500, 500, 0666)
	
	if ui.Name() != "fake" {
		t.Errorf("expected name 'fake', got %v", ui.Name())
	}
	if ui.UID() != 500 {
		t.Errorf("expected uid 500, got %v", ui.UID())
	}
	if ui.GID() != 500 {
		t.Errorf("expected gid 500, got %v", ui.GID())
	}
	if ui.Mode() != 0666 {
		t.Errorf("expected mode 0666, got %v", ui.Mode())
	}
}

// +build windows

package vhd

import (
	"syscall"
	"unsafe"
)

//go:generate go run mksyscall_windows.go -output zvhd.go vhd.go

//sys createVirtualDisk(virtualStorageType *virtualStorageType, path string, virtualDiskAccessMask uint32, securityDescriptor *uintptr, flags uint32, providerSpecificFlags uint32, parameters *createVirtualDiskParameters, o *syscall.Overlapped, handle *syscall.Handle) (err error) [failretval != 0] = VirtDisk.CreateVirtualDisk
//sys openVirtualDisk(virtualStorageType *virtualStorageType, path string, virtualDiskAccessMask uint32, flags uint32, parameters *openVirtualDiskParameters, handle *syscall.Handle) (err error) [failretval != 0] = VirtDisk.OpenVirtualDisk
//sys detachVirtualDisk(handle syscall.Handle, flags uint32, providerSpecificFlags uint32) (err error) [failretval != 0] = VirtDisk.DetachVirtualDisk

type virtualStorageType struct {
	DeviceID uint32
	VendorID [16]byte
}

const virtualDiskAccessNONE uint32 = 0
const virtualDiskAccessATTACHRO uint32 = 65536
const virtualDiskAccessATTACHRW uint32 = 131072
const virtualDiskAccessDETACH uint32 = 262144
const virtualDiskAccessGETINFO uint32 = 524288
const virtualDiskAccessCREATE uint32 = 1048576
const virtualDiskAccessMETAOPS uint32 = 2097152
const virtualDiskAccessREAD uint32 = 851968
const virtualDiskAccessALL uint32 = 4128768
const virtualDiskAccessWRITABLE uint32 = 3276800

const createVirtualDiskFlagNone uint32 = 0
const createVirtualDiskFlagFullPhysicalAllocation uint32 = 1
const createVirtualDiskFlagPreventWritesToSourceDisk uint32 = 2
const createVirtualDiskFlagDoNotCopyMetadataFromParent uint32 = 4

const openVirtualDiskFlagNONE uint32 = 0
const openVirtualDiskFlagNOPARENTS uint32 = 0x1
const openVirtualDiskFlagBLANKFILE uint32 = 0x2
const openVirtualDiskFlagBOOTDRIVE uint32 = 0x4
const openVirtualDiskFlagCACHEDIO uint32 = 0x8
const openVirtualDiskFlagCUSTOMDIFFCHAIN uint32 = 0x10
const openVirtualDiskFlagPARENTCACHEDIO uint32 = 0x20
const openVirtualDiskFlagVHDSETFILEONLY uint32 = 0x40
const openVirtualDiskFlagIGNORERELATIVEPARENTLOCATOR uint32 = 0x80
const openVirtualDiskFlagNOWRITEHARDENING uint32 = 0x100

const vhdWriteCacheModeCacheMetadata uint16 = 0
const vhdWriteCacheModeWriteInternalMetadata uint16 = 1
const vhdWriteCacheModeWriteMetadata uint16 = 2
const vhdWriteCacheModeCommitAll uint16 = 3
const vhdWriteCacheModeDisableFlushing uint16 = 4

type createVersion2 struct {
	UniqueID                 [16]byte // GUID
	MaximumSize              uint64
	BlockSizeInBytes         uint32
	SectorSizeInBytes        uint32
	ParentPath               *uint16 // string
	SourcePath               *uint16 // string
	OpenFlags                uint32
	ParentVirtualStorageType virtualStorageType
	SourceVirtualStorageType virtualStorageType
	ResiliencyGUID           [16]byte // GUID
}

type createVirtualDiskParameters struct {
	Version  uint32 // Must always be set to 2
	Version2 createVersion2
}

type openVersion2 struct {
	GetInfoOnly    bool
	ReadOnly       bool
	ResiliencyGUID [16]byte // GUID
}

type openVirtualDiskParameters struct {
	Version  uint32 // Must always be set to 2
	Version2 openVersion2
}

type storageSetSurfaceCachePolicyRequest struct {
	RequestLevel uint32
	CacheMode    uint16
}

// CreateVhdx will create a simple vhdx file at the given path using default values.
func CreateVhdx(path string, maxSizeInGb, blockSizeInMb uint32) error {
	var defaultType virtualStorageType

	parameters := createVirtualDiskParameters{
		Version: 2,
		Version2: createVersion2{
			MaximumSize:      uint64(maxSizeInGb) * 1024 * 1024 * 1024,
			BlockSizeInBytes: blockSizeInMb * 1024 * 1024,
		},
	}

	var handle syscall.Handle

	if err := createVirtualDisk(
		&defaultType,
		path,
		virtualDiskAccessNONE,
		nil,
		createVirtualDiskFlagNone,
		0,
		&parameters,
		nil,
		&handle); err != nil {
		return err
	}

	if err := syscall.CloseHandle(handle); err != nil {
		return err
	}

	return nil
}

// DetachVhd detaches a VHD attached at the given path.
func DetachVhd(path string) error {
	var (
		defaultType virtualStorageType
		handle      syscall.Handle
	)

	if err := openVirtualDisk(
		&defaultType,
		path,
		virtualDiskAccessDETACH,
		0,
		nil,
		&handle); err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	if err := detachVirtualDisk(handle, 0, 0); err != nil {
		return err
	}
	return nil
}

// VhdWriteCacheModeDisableFlushing sets a VHDs surface cache policy to disable flushing.
// It returns a handle which should be kept open during (for example) container start,
// and restored using VhdWriteCacheModeCacheMetadata.
func VhdWriteCacheModeDisableFlushing(path string) (syscall.Handle, error) {
	handle, err := openVhd(path)
	if err != nil {
		return 0, err
	}
	if err := ioctlStorageSetSurfaceCachePolicy(handle, vhdWriteCacheModeDisableFlushing); err != nil {
		syscall.CloseHandle(handle)
		return 0, err
	}
	return handle, nil
}

// VhdWriteCacheModeCacheMetadata sets a VHDs surface cache policy to cache metadata
func VhdWriteCacheModeCacheMetadata(handle syscall.Handle) error {
	return ioctlStorageSetSurfaceCachePolicy(handle, vhdWriteCacheModeDisableFlushing)
}

// openVhd is a wrapper for getting a handle to a VHD.
func openVhd(path string) (syscall.Handle, error) {
	var (
		defaultType virtualStorageType
		handle      syscall.Handle
	)
	parameters := openVirtualDiskParameters{Version: 2}

	if err := openVirtualDisk(
		&defaultType,
		path,
		virtualDiskAccessNONE,
		openVirtualDiskFlagCACHEDIO|openVirtualDiskFlagIGNORERELATIVEPARENTLOCATOR,
		&parameters,
		&handle); err != nil {
		return 0, err
	}

	return handle, nil
}

func ioctlStorageSetSurfaceCachePolicy(handle syscall.Handle, cacheMode uint16) error {
	request := storageSetSurfaceCachePolicyRequest{
		RequestLevel: 1,
		CacheMode:    cacheMode,
	}
	const storageSetSurfaceCachePolicy uint32 = 0x2d1a10
	var bytesReturned uint32
	return syscall.DeviceIoControl(
		handle,
		storageSetSurfaceCachePolicy,
		(*byte)(unsafe.Pointer(&request)),
		uint32(unsafe.Sizeof(request)),
		nil,
		0,
		&bytesReturned,
		nil)
}

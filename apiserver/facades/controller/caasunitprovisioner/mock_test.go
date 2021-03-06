// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package caasunitprovisioner_test

import (
	"github.com/juju/errors"
	"github.com/juju/testing"
	"gopkg.in/juju/names.v2"

	"github.com/juju/juju/apiserver/facades/controller/caasunitprovisioner"
	"github.com/juju/juju/caas/kubernetes/provider"
	"github.com/juju/juju/controller"
	"github.com/juju/juju/core/application"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/network"
	"github.com/juju/juju/state"
	statetesting "github.com/juju/juju/state/testing"
	"github.com/juju/juju/status"
	"github.com/juju/juju/storage"
	"github.com/juju/juju/storage/poolmanager"
	coretesting "github.com/juju/juju/testing"
)

type mockState struct {
	testing.Stub
	application         mockApplication
	applicationsWatcher *statetesting.MockStringsWatcher
	model               mockModel
	unit                mockUnit
}

func (st *mockState) WatchApplications() state.StringsWatcher {
	st.MethodCall(st, "WatchApplications")
	return st.applicationsWatcher
}

func (st *mockState) Application(name string) (caasunitprovisioner.Application, error) {
	st.MethodCall(st, "Application", name)
	if name != st.application.tag.Id() {
		return nil, errors.NotFoundf("application %v", name)
	}
	return &st.application, st.NextErr()
}

func (st *mockState) FindEntity(tag names.Tag) (state.Entity, error) {
	st.MethodCall(st, "FindEntity", tag)
	if err := st.NextErr(); err != nil {
		return nil, err
	}
	switch tag.(type) {
	case names.ApplicationTag:
		return &st.application, nil
	case names.UnitTag:
		return &st.unit, nil
	default:
		return nil, errors.NotFoundf("%s", names.ReadableString(tag))
	}
}

func (st *mockState) ControllerConfig() (controller.Config, error) {
	st.MethodCall(st, "ControllerConfig")
	return coretesting.FakeControllerConfig(), nil
}

func (st *mockState) Model() (caasunitprovisioner.Model, error) {
	st.MethodCall(st, "Model")
	if err := st.NextErr(); err != nil {
		return nil, err
	}
	return &st.model, nil
}

type mockModel struct {
	testing.Stub
	podSpecWatcher *statetesting.MockNotifyWatcher
}

func (m *mockModel) ModelConfig() (*config.Config, error) {
	m.MethodCall(m, "ModelConfig")
	return config.New(config.UseDefaults, coretesting.FakeConfig())
}

func (m *mockModel) PodSpec(tag names.ApplicationTag) (string, error) {
	m.MethodCall(m, "PodSpec", tag)
	if err := m.NextErr(); err != nil {
		return "", err
	}
	return "spec(" + tag.Id() + ")", nil
}

func (m *mockModel) WatchPodSpec(tag names.ApplicationTag) (state.NotifyWatcher, error) {
	m.MethodCall(m, "WatchPodSpec", tag)
	if err := m.NextErr(); err != nil {
		return nil, err
	}
	return m.podSpecWatcher, nil
}

type mockApplication struct {
	testing.Stub
	life         state.Life
	unitsWatcher *statetesting.MockStringsWatcher

	tag        names.Tag
	units      []caasunitprovisioner.Unit
	ops        *state.UpdateUnitsOperation
	providerId string
	addresses  []network.Address
}

func (*mockApplication) Tag() names.Tag {
	panic("should not be called")
}

func (a *mockApplication) Name() string {
	a.MethodCall(a, "Name")
	return a.tag.Id()
}

func (a *mockApplication) Life() state.Life {
	a.MethodCall(a, "Life")
	return a.life
}

func (a *mockApplication) WatchUnits() state.StringsWatcher {
	a.MethodCall(a, "WatchUnits")
	return a.unitsWatcher
}

func (a *mockApplication) ApplicationConfig() (application.ConfigAttributes, error) {
	a.MethodCall(a, "ApplicationConfig")
	return application.ConfigAttributes{"foo": "bar"}, a.NextErr()
}

func (m *mockApplication) AllUnits() (units []caasunitprovisioner.Unit, err error) {
	return m.units, nil
}

func (m *mockApplication) UpdateUnits(ops *state.UpdateUnitsOperation) error {
	m.ops = ops
	return nil
}

func (m *mockApplication) UpdateCloudService(providerId string, addreses []network.Address) error {
	m.providerId = providerId
	m.addresses = addreses
	return nil
}

var addOp = &state.AddUnitOperation{}

func (m *mockApplication) AddOperation(props state.UnitUpdateProperties) *state.AddUnitOperation {
	m.MethodCall(m, "AddOperation", props)
	return addOp
}

type mockContainerInfo struct {
	state.CloudContainer
	providerId string
}

func (m *mockContainerInfo) ProviderId() string {
	return m.providerId
}

type mockUnit struct {
	testing.Stub
	name          string
	life          state.Life
	containerInfo *mockContainerInfo
}

func (*mockUnit) Tag() names.Tag {
	panic("should not be called")
}

func (u *mockUnit) UnitTag() names.UnitTag {
	return names.NewUnitTag(u.name)
}

func (u *mockUnit) Life() state.Life {
	u.MethodCall(u, "Life")
	return u.life
}

func (m *mockUnit) Name() string {
	return m.name
}

func (m *mockUnit) ContainerInfo() (state.CloudContainer, error) {
	if m.containerInfo == nil {
		return nil, errors.NotFoundf("container info")
	}
	return m.containerInfo, nil
}

func (m *mockUnit) AgentStatus() (status.StatusInfo, error) {
	return status.StatusInfo{Status: status.Allocating}, nil
}

var updateOp = &state.UpdateUnitOperation{}

func (m *mockUnit) UpdateOperation(props state.UnitUpdateProperties) *state.UpdateUnitOperation {
	m.MethodCall(m, "UpdateOperation", props)
	return updateOp
}

var destroyOp = &state.DestroyUnitOperation{}

func (m *mockUnit) DestroyOperation() *state.DestroyUnitOperation {
	m.MethodCall(m, "DestroyOperation")
	return destroyOp
}

type mockStorage struct {
	testing.Stub
}

func (m *mockStorage) StorageInstance(tag names.StorageTag) (state.StorageInstance, error) {
	m.MethodCall(m, "StorageInstance", tag)
	return &mockStorageInstance{
		tag:   tag,
		owner: names.NewUserTag("fred"),
	}, nil
}

func (m *mockStorage) Filesystem(fsTag names.FilesystemTag) (state.Filesystem, error) {
	return nil, errors.NotSupportedf("filesystem")
}

func (m *mockStorage) FilesystemAttachment(hostTag names.Tag, fsTag names.FilesystemTag) (state.FilesystemAttachment, error) {
	m.MethodCall(m, "FilesystemAttachment", hostTag, fsTag)
	return &mockFilesystemAttachment{}, nil
}

func (m *mockStorage) StorageInstanceFilesystem(tag names.StorageTag) (state.Filesystem, error) {
	return &mockFilesystem{}, nil
}

func (m *mockStorage) UnitStorageAttachments(unit names.UnitTag) ([]state.StorageAttachment, error) {
	m.MethodCall(m, "UnitStorageAttachments", unit)
	return []state.StorageAttachment{
		&mockStorageAttachment{
			unit:    names.NewUnitTag("gitlab/0"),
			storage: names.NewStorageTag("data/0"),
		},
	}, nil
}

type mockStorageInstance struct {
	state.StorageInstance
	tag   names.StorageTag
	owner names.Tag
}

func (a *mockStorageInstance) Kind() state.StorageKind {
	return state.StorageKindFilesystem
}

func (a *mockStorageInstance) Tag() names.Tag {
	return a.tag
}

func (a *mockStorageInstance) StorageTag() names.StorageTag {
	return a.tag
}

func (a *mockStorageInstance) Owner() (names.Tag, bool) {
	return a.owner, a.owner != nil
}

type mockStorageAttachment struct {
	state.StorageAttachment
	testing.Stub
	unit    names.UnitTag
	storage names.StorageTag
}

func (a *mockStorageAttachment) StorageInstance() names.StorageTag {
	return a.storage
}

type mockFilesystem struct {
	state.Filesystem
}

func (f *mockFilesystem) Tag() names.Tag {
	return f.FilesystemTag()
}

func (f *mockFilesystem) FilesystemTag() names.FilesystemTag {
	return names.NewFilesystemTag("gitlab/0/0")
}

func (f *mockFilesystem) Volume() (names.VolumeTag, error) {
	return names.VolumeTag{}, state.ErrNoBackingVolume
}

func (f *mockFilesystem) Params() (state.FilesystemParams, bool) {
	return state.FilesystemParams{
		Pool: "k8spool",
		Size: 100,
	}, true
}

type mockFilesystemAttachment struct {
	state.FilesystemAttachment
}

func (f *mockFilesystemAttachment) Params() (state.FilesystemAttachmentParams, bool) {
	return state.FilesystemAttachmentParams{
		Location: "/path/to/here",
		ReadOnly: true,
	}, true
}

type mockStorageProviderRegistry struct {
	testing.Stub
	storage.ProviderRegistry
}

func (m *mockStorageProviderRegistry) StorageProvider(providerType storage.ProviderType) (storage.Provider, error) {
	m.MethodCall(m, "StorageProvider", providerType)
	return nil, errors.NotSupportedf("StorageProvider")
}

type mockStoragePoolManager struct {
	testing.Stub
	poolmanager.PoolManager
}

func (m *mockStoragePoolManager) Get(name string) (*storage.Config, error) {
	m.MethodCall(m, "Get", name)
	return storage.NewConfig(name, provider.K8s_ProviderType, map[string]interface{}{"foo": "bar"})
}

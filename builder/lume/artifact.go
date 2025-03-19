package lume

import (
	"os"
	"path"
)

// packersdk.LumeVMArtifact implementation
type LumeVMArtifact struct {
	VMName string
	// StateData should store data such as GeneratedData
	// to be shared with post-processors
	StateData map[string]interface{}
}

func (*LumeVMArtifact) BuilderId() string {
	return BuilderId
}

func (a *LumeVMArtifact) Files() []string {
	baseDir := a.vmDirPath()
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return []string{}
	}
	result := make([]string, len(entries))
	for index, entry := range entries {
		result[index] = path.Join(entry.Name())
	}
	return result
}

func (a *LumeVMArtifact) Id() string {
	return a.VMName
}

func (a *LumeVMArtifact) String() string {
	return a.VMName
}

func (a *LumeVMArtifact) State(name string) interface{} {
	return a.StateData[name]
}

func (a *LumeVMArtifact) Destroy() error {
	return os.RemoveAll(a.vmDirPath())
}

func (a *LumeVMArtifact) vmDirPath() string {
	return PathInLumeHome("vms", a.VMName)
}

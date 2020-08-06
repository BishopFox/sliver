package memprofile

// MemAllocator represents a memory allocator
type MemAllocator interface {
	// Allocate defines the memory allocation routine
	Allocate() error
}

// MemExecutor represents the memory executor
type MemExecutor interface {
	// Execute defines the execution routine
	Execute() error
}

// MemProfile represents a memory injection profile
type MemProfile struct {
	Allocator MemAllocator
	Executor  MemExecutor
	Name      string
}

// Inject allocates and executes the profile payload
// func (m *MemProfile) Inject() error {
// 	if err := m.Allocator.Allocate(); err != nil {
// 		return err
// 	}
// 	return m.Executor.Execute()
// }

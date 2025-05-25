package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDatabase is a mock implementation of the Database interface
type MockDatabase struct {
	mock.Mock
}

// GetAllBoilerplates mocks the GetAllBoilerplates method
func (m *MockDatabase) GetAllBoilerplates() (map[string]*Boilerplate, error) {
	args := m.Called()

	// Handle nil return case
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(map[string]*Boilerplate), args.Error(1)
}

// GetBoilerplateByName mocks the GetBoilerplateByName method
func (m *MockDatabase) GetBoilerplateByName(name string) (*Boilerplate, error) {
	args := m.Called(name)

	// Handle nil return case
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Boilerplate), args.Error(1)
}

// CreateBoilerplate mocks the CreateBoilerplate method
func (m *MockDatabase) CreateBoilerplate(boilerplate *Boilerplate) error {
	args := m.Called(boilerplate)
	return args.Error(0)
}

// UpdateBoilerplate mocks the UpdateBoilerplate method
func (m *MockDatabase) UpdateBoilerplate(boilerplate *Boilerplate) error {
	args := m.Called(boilerplate)
	return args.Error(0)
}

// DeleteBoilerplate mocks the DeleteBoilerplate method
func (m *MockDatabase) DeleteBoilerplate(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// IncBoilerplateCount mocks the IncBoilerplateCount method
func (m *MockDatabase) IncBoilerplateCount(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// Close mocks the Close method
func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestSQLiteDatabase_Integration(t *testing.T) {
	// Create a temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, defaultDatabaseFileName)

	// Create new SQLite database
	db, err := NewSQLiteDatabase(dbPath)
	require.NoError(t, err, "Failed to create database")
	defer db.Close()

	// Test data - several boilerplates to add
	testBoilerplates := []*Boilerplate{
		{
			Name:  "greeting",
			Value: "Hello {{name}}, welcome to our service!",
			Count: 0,
		},
		{
			Name:  "farewell",
			Value: "Goodbye [[user]], see you soon!",
			Count: 2,
		},
		{
			Name:  "reminder",
			Value: "Don't forget about {{event}} on {{date}}",
			Count: 5,
		},
	}

	// Add several values to the database
	t.Run("Create boilerplates", func(t *testing.T) {
		for _, bp := range testBoilerplates {
			err := db.CreateBoilerplate(bp)
			require.NoError(t, err, "Failed to create boilerplate: %s", bp.Name)
		}
	})

	// Retrieve them and check their values
	t.Run("Retrieve and verify boilerplates", func(t *testing.T) {
		// Get all boilerplates
		allBoilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err, "Failed to get all boilerplates")

		// Check we have the right number
		assert.Len(t, allBoilerplates, 3, "Should have 3 boilerplates")

		// Verify each boilerplate
		for _, expected := range testBoilerplates {
			actual, exists := allBoilerplates[expected.Name]
			require.True(t, exists, "Boilerplate %s should exist", expected.Name)

			assert.Equal(t, expected.Name, actual.Name, "Name should match")
			assert.Equal(t, expected.Value, actual.Value, "Value should match")
			assert.Equal(t, expected.Count, actual.Count, "Count should match")
		}

		// Test individual retrieval
		greetingBP, err := db.GetBoilerplateByName("greeting")
		require.NoError(t, err, "Failed to get greeting boilerplate")
		require.NotNil(t, greetingBP, "Greeting boilerplate should not be nil")

		assert.Equal(t, "greeting", greetingBP.Name)
		assert.Equal(t, "Hello {{name}}, welcome to our service!", greetingBP.Value)
		assert.Equal(t, 0, greetingBP.Count)

		// Test retrieving non-existent boilerplate
		nonExistent, err := db.GetBoilerplateByName("does-not-exist")
		require.Error(t, err, "Should error on non-existent boilerplate")
		assert.Nil(t, nonExistent, "Non-existent boilerplate should return nil")
	})

	// Increment a count
	t.Run("Increment boilerplate count", func(t *testing.T) {
		// Increment the greeting boilerplate count (should go from 0 to 1)
		err := db.IncBoilerplateCount("greeting")
		require.NoError(t, err, "Failed to increment greeting count")

		// Increment it again (should go from 1 to 2)
		err = db.IncBoilerplateCount("greeting")
		require.NoError(t, err, "Failed to increment greeting count again")

		// Increment farewell count (should go from 0 to 1)
		err = db.IncBoilerplateCount("farewell")
		require.NoError(t, err, "Failed to increment farewell count")

		// Try to increment non-existent boilerplate
		err = db.IncBoilerplateCount("does-not-exist")
		require.Error(t, err, "Should error when incrementing non-existent boilerplate")
	})

	// Check that boilerplate counts are correct
	t.Run("Verify updated counts", func(t *testing.T) {
		// Check greeting count (should be 2 now)
		greetingBP, err := db.GetBoilerplateByName("greeting")
		require.NoError(t, err, "Failed to get greeting boilerplate")
		require.NotNil(t, greetingBP, "Greeting boilerplate should exist")
		assert.Equal(t, 2, greetingBP.Count, "Greeting count should be 2 after incrementing twice")

		// Check farewell count (should be 3 now)
		farewellBP, err := db.GetBoilerplateByName("farewell")
		require.NoError(t, err, "Failed to get farewell boilerplate")
		require.NotNil(t, farewellBP, "Farewell boilerplate should exist")
		assert.Equal(t, 3, farewellBP.Count, "Farewell count should be 3 after incrementing once")

		// Check reminder count (should still be 5, unchanged)
		reminderBP, err := db.GetBoilerplateByName("reminder")
		require.NoError(t, err, "Failed to get reminder boilerplate")
		require.NotNil(t, reminderBP, "Reminder boilerplate should exist")
		assert.Equal(t, 5, reminderBP.Count, "Reminder count should still be 5")

		// Verify through GetAllBoilerplates as well
		allBoilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err, "Failed to get all boilerplates")

		assert.Equal(t, 2, allBoilerplates["greeting"].Count, "Greeting count should be 2 in GetAll")
		assert.Equal(t, 3, allBoilerplates["farewell"].Count, "Farewell count should be 3 in GetAll")
		assert.Equal(t, 5, allBoilerplates["reminder"].Count, "Reminder count should be 5 in GetAll")
	})

	// Test additional operations
	t.Run("Test update and delete operations", func(t *testing.T) {
		// Update a boilerplate
		updatedGreeting := &Boilerplate{
			Name:  "greeting",
			Value: "Hi {{name}}, welcome back!",
			Count: 10, // Update count as well
		}

		err := db.UpdateBoilerplate(updatedGreeting)
		require.NoError(t, err, "Failed to update greeting boilerplate")

		// Verify the update
		retrievedBP, err := db.GetBoilerplateByName("greeting")
		require.NoError(t, err, "Failed to retrieve updated boilerplate")
		assert.Equal(t, "Hi {{name}}, welcome back!", retrievedBP.Value, "Value should be updated")
		assert.Equal(t, 10, retrievedBP.Count, "Count should be updated")

		// Delete a boilerplate
		err = db.DeleteBoilerplate("reminder")
		require.NoError(t, err, "Failed to delete reminder boilerplate")

		// Verify deletion
		deletedBP, err := db.GetBoilerplateByName("reminder")
		require.Error(t, err, "Should error when getting deleted boilerplate")
		assert.Nil(t, deletedBP, "Deleted boilerplate should return nil")

		// Verify we now have 2 boilerplates instead of 3
		allBoilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err, "Failed to get all boilerplates after deletion")
		assert.Len(t, allBoilerplates, 2, "Should have 2 boilerplates after deletion")
	})

	// Close the database (this will be called by defer as well, but testing explicitly)
	t.Run("Close database", func(t *testing.T) {
		err := db.Close()
		assert.NoError(t, err, "Failed to close database")
	})
}

// Additional test for edge cases
func TestSQLiteDatabase_EdgeCases(t *testing.T) {
	// Create a temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, defaultDatabaseFileName)

	db, err := NewSQLiteDatabase(dbPath)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Test empty database", func(t *testing.T) {
		// Get all from empty database
		boilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err)
		assert.Empty(t, boilerplates, "Empty database should return empty map")

		// Get non-existent boilerplate
		bp, err := db.GetBoilerplateByName("nonexistent")
		require.Error(t, err)
		assert.Nil(t, bp, "Non-existent boilerplate should return nil")
	})

	t.Run("Test duplicate creation", func(t *testing.T) {
		bp := &Boilerplate{
			Name:  "duplicate",
			Value: "test value",
			Count: 0,
		}

		// Create first time - should succeed
		err := db.CreateBoilerplate(bp)
		require.NoError(t, err)

		// Create again with same name - should fail
		err = db.CreateBoilerplate(bp)
		require.Error(t, err, "Creating duplicate boilerplate should fail")
	})
}

func TestSQLiteDatabase_PersistenceEmpty(t *testing.T) {
	// Create a temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, defaultDatabaseFileName)

	// Create empty database and close
	t.Run("Create empty database", func(t *testing.T) {
		db, err := NewSQLiteDatabase(dbPath)
		require.NoError(t, err)

		// Verify it's empty
		boilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err)
		assert.Empty(t, boilerplates)

		err = db.Close()
		require.NoError(t, err)
	})

	// Reopen and verify still empty
	t.Run("Verify empty database persists", func(t *testing.T) {
		db, err := NewSQLiteDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		boilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err)
		assert.Empty(t, boilerplates, "Reopened empty database should still be empty")
	})
}

func TestSQLiteDatabase_Persistence(t *testing.T) {
	// Create a temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, defaultDatabaseFileName)

	initialBoilerplates := []*Boilerplate{
		{
			Name:  "greeting",
			Value: "Hello {{name}}, welcome to our service!",
			Count: 0,
		},
		{
			Name:  "farewell",
			Value: "Goodbye [[user]], see you soon!",
			Count: 2,
		},
		{
			Name:  "reminder",
			Value: "Don't forget about {{event}} on {{date}}",
			Count: 5,
		},
	}

	// Phase 1: Create, insert, modify, and close
	var modifiedBoilerplates []*Boilerplate
	t.Run("Create database, insert and modify values", func(t *testing.T) {
		db, err := NewSQLiteDatabase(dbPath)
		require.NoError(t, err)

		// Insert initial boilerplates
		for _, bp := range initialBoilerplates {
			err := db.CreateBoilerplate(bp)
			require.NoError(t, err)
		}

		// Modify some boilerplates
		modifiedBoilerplates = make([]*Boilerplate, len(initialBoilerplates))
		for i, bp := range initialBoilerplates {
			// Create a modified copy
			modified := &Boilerplate{
				Name:  bp.Name,
				Value: bp.Value + " [MODIFIED]",
				Count: bp.Count + 10,
			}

			// Increment count a few times
			for j := 0; j < 3; j++ {
				err := db.IncBoilerplateCount(bp.Name)
				require.NoError(t, err)
				modified.Count++
			}

			// Update the boilerplate
			err := db.UpdateBoilerplate(modified)
			require.NoError(t, err)

			// Store expected final state
			modifiedBoilerplates[i] = &Boilerplate{
				Name:  modified.Name,
				Value: modified.Value,
				Count: modified.Count,
			}
		}

		err = db.Close()
		require.NoError(t, err)
	})

	// Phase 2: Reopen and verify modifications persisted
	t.Run("Verify modifications persisted after reopening", func(t *testing.T) {
		db, err := NewSQLiteDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Verify all modifications persisted
		for _, expected := range modifiedBoilerplates {
			actual, err := db.GetBoilerplateByName(expected.Name)
			require.NoError(t, err)
			require.NotNil(t, actual)

			assert.Equal(t, expected.Name, actual.Name,
				"Modified name should persist")
			assert.Equal(t, expected.Value, actual.Value,
				"Modified value should persist")
			assert.Equal(t, expected.Count, actual.Count,
				"Modified count should persist")
		}

		// Verify through GetAllBoilerplates as well
		allBoilerplates, err := db.GetAllBoilerplates()
		require.NoError(t, err)
		assert.Len(t, allBoilerplates, len(modifiedBoilerplates))

		for _, expected := range modifiedBoilerplates {
			actual := allBoilerplates[expected.Name]
			require.NotNil(t, actual)
			assert.Equal(t, expected.Value, actual.Value)
			assert.Equal(t, expected.Count, actual.Count)
		}
	})
}

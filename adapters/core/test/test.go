// Package test is the core test suite for all database drivers.
// A conformant implementation should pass all tests.
//
// This test suite is not meant to be run on it's own. The individual
// driver folders have their own dyndao_test.go which bootstraps this
// code.

/*
	TODO: I need to research Go testing patterns more thoroughly.
	For database driver testing, it would appear that panic()
	would be more suitable than t.Fatal() or t.Fatalf()...

	Then I would get a stacktrace, and program execution would halt,
	making it much more apparent that something had gone wrong.
*/

package test

import (
	"context"
	"database/sql"
	"os"
	"reflect"
	"time"
	//"fmt"

	"testing"

	"github.com/rbastic/dyndao/object"
	"github.com/rbastic/dyndao/orm"
	"github.com/rbastic/dyndao/schema"
	"github.com/rbastic/dyndao/schema/test/mock"

	sg "github.com/rbastic/dyndao/sqlgen"
)

type FnGetDB func() *sql.DB // function type for GetDB
type FnGetSG func() *sg.SQLGenerator // function type for GetSQLGenerator

var (
	GetDB     FnGetDB
	getSQLGen FnGetSG
)

// getDefaultContext returns the standard context used by the test package.
func getDefaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

func fatalIf(err error) {
	if err != nil {
		panic(err)
	}
}

func PingCheck(t *testing.T, db *sql.DB) {
	ctx, cancel := getDefaultContext()
	err := db.PingContext(ctx)
	cancel()
	fatalIf(err)
}

func dirtyTest(obj * object.Object) {
	if obj.IsDirty() {
		panic("system claims object is not saved")
	}
}

func Test(t *testing.T, getDBFn FnGetDB, getSGFN FnGetSG) {
	// Set our functions locally
	GetDB = getDBFn
	getSQLGen = getSGFN

	db := GetDB()
	if db == nil {
		t.Fatal("dyndao: core/test/Test: GetDB() returned a nil value.")
	}
	defer func() {
		err := db.Close()
		fatalIf(err)
	}()

	// Bootstrap the db, run the test suite, drop tables
	t.Run("TestPingCheck", func(t *testing.T) {
		PingCheck(t, db)
	})

	if os.Getenv("DROP_TABLES") != "" {
		t.Run("TestDropTables", func(t *testing.T) {
			TestDropTables(t, db)
		})
	}

	t.Run("TestCreateTables", func(t *testing.T) {
		TestCreateTables(t, db)
	})

	TestSuiteNested(t, db)

	t.Run("TestDropTables", func(t *testing.T) {
		TestDropTables(t, db)
	})
}

func TestCreateTables(t *testing.T, db *sql.DB) {
	sch := mock.NestedSchema()
	o := orm.New(getSQLGen(), sch, db)

	ctx, cancel := getDefaultContext()
	err := o.CreateTables(ctx)
	cancel()
	if err != nil {
		panic(err)
	}
}

func TestDropTables(t *testing.T, db *sql.DB) {
	sch := mock.NestedSchema()
	o := orm.New(getSQLGen(), sch, db)

	ctx, cancel := getDefaultContext()
	err := o.DropTables(ctx)
	cancel()
	fatalIf(err)
}

func validateMock(t * testing.T, obj * object.Object) {
	// Validate that we correctly fleshened the primary key
	t.Run("ValidatePerson/ID", func(t *testing.T) {
		validatePersonID(t, obj)

		// TODO: Make sure we saved the Address with a person id also
	})
	t.Run("ValidatePerson/NullText", func(t *testing.T) {
		validateNullText(t, obj)
	})

	t.Run("ValidatePerson/NullInt", func(t *testing.T) {
		validateNullInt(t, obj)
	})
	t.Run("ValidatePerson/NullVarchar", func(t *testing.T) {
		validateNullVarchar(t, obj)
	})
	t.Run("ValidatePerson/NullBlob", func(t *testing.T) {
		validateNullBlob(t, obj)
	})

	// Validate that we correctly saved the children
	t.Run("ValidateChildrenSaved", func(t *testing.T) {
		validateChildrenSaved(t, obj)
	})
}

func TestSuiteNested(t *testing.T, db *sql.DB) {
	sch := mock.NestedSchema()            // Use mock test schema
	o := orm.New(getSQLGen(), sch, db)    // Setup our ORM
	obj := mock.DefaultPersonWithAddress() // Construct our default mock object

	// Save our default object
	t.Run("SaveMockObject", func(t *testing.T) {
		saveMockObject(t, &o, obj)
	})

	validateMock(t, obj)

	// Test second additional Save to ensure that we don't save
	// the object twice needlessly... This caught a silly bug early on.
	t.Run("TestAdditionalSave", func(t *testing.T) {
		ctx, cancel := getDefaultContext()
		rowsAff, err := o.Save(ctx, nil, obj)
		cancel()
		fatalIf(err)
		if rowsAff != 0 {
			t.Fatal("rowsAff should be zero the second time")
		}
	})

	// Now, trigger an update.
	t.Run("TestUpdateObject", func(t *testing.T) {
		// Changing the name should become an
		// update when we save
		obj.Set("Name", "Joe")
		// Test saving the object
		testSave(&o, t, obj)
	})

	t.Run("Retrieve", func(t *testing.T) {
		// test retrieving the parent, given a child object
		testRetrieve(&o, t, sch)
	})

	t.Run("RetrieveMany", func(t *testing.T) {
		// test multiple retrieve
		testRetrieveMany(&o, t, mock.PeopleObjectType)
	})

	t.Run("FleshenChildren", func(t *testing.T) {
		// try fleshen children on person id 1
		testFleshenChildren(&o, t, mock.PeopleObjectType)
	})

	t.Run("GetParentsViaChild", func(t *testing.T) {
		// test retrieving multiple parents, given a single child object
		testGetParentsViaChild(&o, t)
	})
}

func saveMockObject(t *testing.T, o *orm.ORM, obj *object.Object) {
	ctx, cancel := getDefaultContext()
	rowsAff, err := o.SaveAll(ctx, obj)
	cancel()
	fatalIf(err)

	dirtyTest(obj)

	if rowsAff == 0 {
		t.Fatal("Rows affected shouldn't be zero initially")
	}
}

func validatePersonID(t *testing.T, obj *object.Object) {
	personID, err := obj.GetIntAlways("PersonID")
	if err != nil {
		t.Fatalf("Had problems retrieving PersonID as int: %s", err.Error())
	}
	if personID != 1 {
		if personID >= 2 {
			t.Fatal("Tests are not in a ready state. Pre-existing data is present.")
		}
		t.Fatalf("PersonID has the wrong value, has value %d", personID)
	}
}

func validateNullText(t *testing.T, obj *object.Object) {
	if !obj.ValueIsNULL(obj.Get("NullText")) {
		t.Fatal("validateNullText expected NULL value")
	}
}

func validateNullInt(t *testing.T, obj *object.Object) {
	if !obj.ValueIsNULL(obj.Get("NullInt")) {
		t.Fatal("validateNullInt: expected NULL value")
	}
}

func validateNullVarchar(t *testing.T, obj *object.Object) {
	if !obj.ValueIsNULL(obj.Get("NullVarchar")) {
		t.Fatal("validateNullVarchar: expected NULL value")
	}
}

func validateNullBlob(t *testing.T, obj *object.Object) {
	if !obj.ValueIsNULL(obj.Get("NullBlob")) {
		t.Fatal("validateNullBlob: expected NULL value")
	}
}

func validateChildrenSaved(t *testing.T, obj *object.Object) {
	for _, childs := range obj.Children {
		for _, v := range childs {
			dirtyTest(v)

			addrID := v.Get("AddressID").(int64)
			if addrID < 1 {
				t.Fatal("AddressID was not what we expected")
			}
		}
	}
}

func testSave(o *orm.ORM, t *testing.T, obj *object.Object) {
	ctx, cancel := getDefaultContext()
	rowsAff, err := o.Save(ctx, nil, obj)
	cancel()

	fatalIf(err)
	if rowsAff == 0 {
		t.Fatalf("rowsAff should not be zero")
	}

	dirtyTest(obj)
}

func testRetrieve(o *orm.ORM, t *testing.T, sch *schema.Schema) {
	queryVals := map[string]interface{}{
		"PersonID": 1,
	}
	// refleshen our object
	ctx, cancel := getDefaultContext()
	latestJoe, err := o.Retrieve(ctx, mock.PeopleObjectType, queryVals)
	cancel()
	if err != nil {
		t.Fatal("retrieve failed: " + err.Error())
	}
	if latestJoe == nil {
		t.Fatal("LatestJoe Should not be nil!")
	}
	// TODO: Do a common refactor on this sort of code
	nameStr, err := latestJoe.GetStringAlways("Name")
	fatalIf(err)
	if latestJoe.Get("PersonID").(int64) != 1 || nameStr != "Joe" {
		t.Fatal("latestJoe does not match expectations")
	}
}

func testRetrieveMany(o *orm.ORM, t *testing.T, rootTable string) {
	// insert another object
	nobj := object.New(rootTable)
	nobj.Set("Name", "Joe")
	{
		ctx, cancel := getDefaultContext()
		rowsAff, err := o.SaveAll(ctx, nobj)
		cancel()
		if err != nil {
			t.Fatal("Save:" + err.Error())
		}
		dirtyTest(nobj)

		if rowsAff == 0 {
			t.Fatal("Rows affected shouldn't be zero initially")
		}
	}

	// try a full table scan
	ctx, cancel := getDefaultContext()
	all, err := o.RetrieveMany(ctx, rootTable, make(map[string]interface{}))
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatal("Should only be 2 rows inserted")
	}

	car := all[0]
	cdr := all[1]

	{
		if car.Get("Name") == cdr.Get("Name") && car.Get("PersonID") != cdr.Get("PersonID") {
			// pass
		} else {
			t.Fatal("objects weren't what we expected? they appear to be the same?")
		}
	}

	if reflect.DeepEqual(car, cdr) {
		t.Fatal("Objects matched, this was not expected")
	}
}

func testGetParentsViaChild(o *orm.ORM, t *testing.T) {
	// Configure our database query
	queryVals := make(map[string]interface{})
	queryVals["PersonID"] = 1
	// Retrieve a single child object
	ctx, cancel := getDefaultContext()
	childObj, err := o.Retrieve(ctx, "addresses", queryVals)
	cancel()

	fatalIf(err)
	if childObj == nil {
		t.Fatal("testGetParentsViaChild: childObj was nil")
	}
	// A silly test, but verifies basic assumptions
	if childObj.Type != "addresses" {
		t.Fatal("Unknown child object retrieved", childObj)
	}

	// Retrieve the parents of that child object
	ctx, cancel = getDefaultContext()
	objs, err := o.GetParentsViaChild(ctx, childObj)
	fatalIf(err)

	// Validate expected data
	if len(objs) != 1 {
		t.Fatal("Unknown length of objs, expected 1, got ", len(objs))
	}
	obj := objs[0]

	nameStr, err := obj.GetStringAlways("Name")
	fatalIf(err)
	if obj.Get("PersonID").(int64) != 1 || nameStr != "Joe" {
		t.Fatal("Object values differed from expectations", obj)
	}
}

func testFleshenChildren(o *orm.ORM, t *testing.T, rootTable string) {
	ctx, cancel := getDefaultContext()
	obj, err := o.Retrieve(ctx, rootTable, map[string]interface{}{
		"PersonID": 1,
	})
	cancel()
	fatalIf(err)

	if obj == nil {
		t.Fatal("object should not be nil")
	}

	{
		ctx, cancel := getDefaultContext()
		fleshened, err := o.FleshenChildren(ctx, obj)
		cancel()
		fatalIf(err)

		if fleshened.Type != mock.PeopleObjectType {
			t.Fatal("fleshened object has wrong type, expected", mock.AddressesObjectType)
		}
		if fleshened.Children[mock.AddressesObjectType] == nil {
			t.Fatal("expected Addresses children")
		}
		address1, err := fleshened.Children["addresses"][0].GetStringAlways("Address1")
		if err != nil {
			panic(err)
		}
		expectedStr := "Test"
		if address1 != "Test" {
			t.Fatalf("expected %s for 'Address1', address1 was %s", expectedStr, address1)
		}
	}
}

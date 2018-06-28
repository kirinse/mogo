package bongo

import (
	"testing"

	"github.com/globalsign/mgo"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_, _ = Connect(&Config{
		Database:         "bongotest",
		ConnectionString: "localhost",
	})
}

// BadDocument is not a valid document because it doesn't have
// the DocumentModel field
type BadDocument struct {
	Name    string
	Surname string
}

// DocumentWithModel is a valid document because it has the
// DocumentModel field and also define the collection name
type DocumentWithModel struct {
	DocumentModel `bson:",inline" coll:"test"`
	Name          string
	Surname       string
}

// DocumentWithModelAndIdx is a valid document because it has the
// DocumentModel field, define the collection name. Also it defines
// some index that will be stored in the collection
type DocumentWithModelAndIdx struct {
	DocumentModel `bson:",inline" coll:"test" idx:"{name,surname},unique"`
	Name          string `idx:"{name},unique,sparse"`
	Surname       string
}

type DocumentWithChildren struct {
	DocumentModel `bson:",inline" coll:"parent-collection" idx:"{name,surname},unique"`
	Name          string `idx:"{name},unique,sparse" coll:"parent-colleciton"` // WARN call is used outside DM
	Surname       string
	Childs        []RefField `ref:"DocumentChild"`
	Child         RefField   `ref:"DocumentChild"`
}

type DocumentWithChildrenNoRef struct {
	DocumentModel `bson:",inline" coll:"parent-collection" idx:"{name,surname},unique"`
	Name          string `idx:"{name},unique,sparse" coll:"parent-colleciton"`
	Surname       string
	Child         []RefField // This field should have ref tag
}
type DocumentChild struct {
	DocumentModel `bson:",inline" coll:"child-collection" idx:"{name,surname},unique"`
	Name          string `idx:"{name},unique,sparse"`
	Surname       string
}

func TestNewDocument(t *testing.T) {
	Convey("should create a new document if document is valid or panic if document is invalid", t, func() {
		doc := NewDocument(DocumentWithModelAndIdx{
			Name:    "MyName",
			Surname: "MySurname",
		}).(*DocumentWithModelAndIdx)

		So(doc.Name, ShouldEqual, "MyName")
		So(doc.Surname, ShouldEqual, "MySurname")

		So(func() { _ = NewDocument(BadDocument{}) }, ShouldPanic)
		So(func() { _ = NewDocument(DocumentWithModel{}).(*DocumentWithModel) }, ShouldNotPanic)
	})
}

func TestNewDocumentWithChildren(t *testing.T) {
	Convey("should create a new document if document is valid or panic if document is invalid", t, func() {
		So(func() {
			ModelRegistry.Register(DocumentWithChildren{},
				DocumentWithChildrenNoRef{},
				DocumentChild{})
		}, ShouldPanic)

		doc := NewDocument(DocumentWithChildren{
			Name:    "MyName",
			Surname: "MySurname",
		}).(*DocumentWithChildren)

		So(doc.Name, ShouldEqual, "MyName")
		So(doc.Surname, ShouldEqual, "MySurname")

		So(func() { _ = NewDocument(BadDocument{}) }, ShouldPanic)
		So(func() { _ = NewDocument(DocumentWithModel{}).(*DocumentWithModel) }, ShouldNotPanic)

		So(func() {
			_ = NewDocument(DocumentWithChildrenNoRef{
				Name:    "MyName",
				Surname: "MySurname",
			}).(*DocumentWithChildrenNoRef)
		}, ShouldPanic)
	})
}

func TestGetParsedIndex(t *testing.T) {
	ModelRegistry.Register(DocumentWithModelAndIdx{})
	Convey("should return the parsed indexes as defined in idx tag", t, func() {
		doc := NewDocument(DocumentWithModelAndIdx{}).(*DocumentWithModelAndIdx)
		pi := doc.GetParsedIndex("Name")
		So(pi, ShouldResemble, []ParsedIndex{
			ParsedIndex{[]string{"name"}, []string{"unique", "sparse"}, 0, false}})
		pi = doc.GetParsedIndex("Boh")
		So(pi, ShouldBeNil)
		rm := make(map[string][]ParsedIndex, 0)
		rm["DocumentModel"] = []ParsedIndex{ParsedIndex{[]string{"name", "surname"}, []string{"unique"}, 1, false}}
		rm["Name"] = []ParsedIndex{ParsedIndex{[]string{"name"}, []string{"unique", "sparse"}, 0, false}}
		rm["Surname"] = nil
		mi := doc.GetAllParsedIndex()
		So(mi, ShouldResemble, rm)
	})
}

func TestGetIndex(t *testing.T) {
	Convey("should return a  []*mgo.Index from the []ParsedIndex built from idx tag of the Name field", t, func() {
		doc := NewDocument(DocumentWithModelAndIdx{}).(*DocumentWithModelAndIdx)
		idx := doc.GetIndex("Name")
		So(len(idx), ShouldBeGreaterThan, 0)
		mi := &mgo.Index{
			Key:    []string{"name"},
			Unique: true,
			Sparse: true,
		}
		So(idx[0], ShouldResemble, mi)
	})
}

func TestGetAllIndex(t *testing.T) {
	Convey("should return a []*mgo.Index from the []ParsedIndex built from idx tags of all fields", t, func() {
		doc := NewDocument(&DocumentWithModelAndIdx{
			Name: "MyFirst",
		}).(*DocumentWithModelAndIdx)
		idx := doc.GetAllIndex()
		So(len(idx), ShouldBeGreaterThan, 0)
		mi := &mgo.Index{
			Key:    []string{"name"},
			Unique: true,
			Sparse: true,
		}
		So(idx[1], ShouldResemble, mi)
	})
}
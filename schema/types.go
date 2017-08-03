package schema

// Schema is the metadata container for a schema definition
type Schema struct {
	Tables map[string]*Table `json:"Tables"`
	// For get ops
	TableAliases map[string]string `json:"TableAliases"`
}

// Table is the metadata container for a SQL table definition
type Table struct {
	MultiKey  bool   `json:"MultiKey"` // Use Primary or Primary + ForeignKeys
	Primary   string `json:"Primary"`
	Name      string `json:"Name"`
	AliasName string `json:"AliasName"` // Combined with AliasName - for set op	s

	// MultiKey must be set to true if a table has
	// foreign keys.
	ForeignKeys []string `json:"ForeignKeys"`
	// Fields is the column definitions for the SQL table
	Fields       map[string]*Field `json:"Fields"`
	FieldAliases map[string]string `json:"FieldAliases"`

	EssentialFields []string `json:"EssentialFields"`

	ParentTables []string               `json:"ParentTables"`
	Children     map[string]*ChildTable `json:"Children"`

	// YAGNI?
	// TODO: ChildrenInsertionOrder?
	// TODO: DeletionOrder?
}

// GetTableName returns either ourDefault or the override string. It is assumed
// that you'll pass in the table key (schema.Tables[key]) and the table name
// (schema.Tables[key].Name), so that this wrapper function can decide.
func GetTableName(override string, ourDefault string) string {
	var name string
	if override != "" {
		name = override
	} else {
		name = ourDefault
	}
	return name
}

// Field represents a single column in a SQL table
type Field struct {
	AllowNull    bool   `json:"AllowNull"`
	IsNumber     bool   `json:"IsNumber"`
	IsIdentity   bool   `json:"IsIdentity"`
	IsForeignKey bool   `json:"IsForeignKey"`
	IsUnique     bool   `json:"IsUnique"`
	Length       int    `json:"Length"`
	Name         string `json:"Name"`
	DefaultValue string `json:"DefaultValue"` // Converts to integer if IsNumber is set
	DBType       string `json:"DBType"`
	Source       string `json:"Source"` // Could be JSON source, could be something else...
}

// ChildTable represents a relationship between a parent table
// and a child table
type ChildTable struct {
	ParentTable string `json:"ParentTable"`

	MultiKey     bool   `json:"MultiKey"`
	LocalField   string `json:"LocalField"`
	ForeignField string `json:"ForeignField"`

	LocalFields   []string `json:"LocalFields"`
	ForeignFields []string `json:"ForeignFields"`
}

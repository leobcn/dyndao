package schema

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestJSONMarshalUnmarshal(t *testing.T) {
	sch := MockBasicSchema()
	buf, err := json.Marshal(sch)
	if err != nil {
		t.Fatal(err)
	}
	// TODO: Fix this test.
	fmt.Println("Marshalled buf=", string(buf))

	err = json.Unmarshal(buf, &sch)
	if err != nil {
		t.Fatal(err)
	}
	// TODO: Fix this test.
	fmt.Println("Unmarshalled sch=", sch)
}

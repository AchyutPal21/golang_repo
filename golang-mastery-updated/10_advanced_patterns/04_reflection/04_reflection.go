// FILE: 10_advanced_patterns/04_reflection.go
// TOPIC: reflect Package — TypeOf, ValueOf, struct tags, setting values
//
// Run: go run 10_advanced_patterns/04_reflection.go

package main

import (
	"fmt"
	"reflect"
	"strconv"
)

type User struct {
	Name  string `json:"name" validate:"required"`
	Age   int    `json:"age"  validate:"min=0,max=150"`
	Email string `json:"email"`
	score int    // unexported — reflection can see it but can't set it
}

// simpleValidator demonstrates how json/yaml/validate libraries work internally
func simpleValidator(v interface{}) []string {
	var errors []string
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		value := rv.Field(i)
		tag := field.Tag.Get("validate")

		if tag == "required" && value.String() == "" {
			errors = append(errors, fmt.Sprintf("field %q is required", field.Name))
		}
		if tag != "" && field.Type.Kind() == reflect.Int {
			// Parse min= and max= from tag
			var min, max int = -1, -1
			fmt.Sscanf(tag, "min=%d,max=%d", &min, &max)
			v := int(value.Int())
			if min >= 0 && v < min {
				errors = append(errors, fmt.Sprintf("field %q: %d < min %d", field.Name, v, min))
			}
			if max >= 0 && v > max {
				errors = append(errors, fmt.Sprintf("field %q: %d > max %d", field.Name, v, max))
			}
		}
	}
	return errors
}

// structToMap converts struct fields to map[string]string using json tags
func structToMap(v interface{}) map[string]string {
	result := make(map[string]string)
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() { continue }
		key := field.Tag.Get("json")
		if key == "" { key = field.Name }
		result[key] = fmt.Sprintf("%v", rv.Field(i).Interface())
	}
	return result
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: reflect Package")
	fmt.Println("════════════════════════════════════════")

	// ── reflect.TypeOf / ValueOf ──────────────────────────────────────────
	fmt.Println("\n── TypeOf / ValueOf ──")
	x := 42
	fmt.Printf("  reflect.TypeOf(x):  %v\n", reflect.TypeOf(x))
	fmt.Printf("  reflect.ValueOf(x): %v\n", reflect.ValueOf(x))
	fmt.Printf("  Kind: %v\n", reflect.ValueOf(x).Kind())

	// Kind vs Type:
	// Kind = the fundamental type category (int, struct, slice, ptr, ...)
	// Type = the specific type (int, User, []string, *int, ...)
	type MyInt int
	var m MyInt = 5
	fmt.Printf("  MyInt — Type: %v, Kind: %v\n", reflect.TypeOf(m), reflect.ValueOf(m).Kind())

	// ── Inspecting structs ────────────────────────────────────────────────
	fmt.Println("\n── Struct field inspection ──")
	u := User{Name: "Alice", Age: 30, Email: "alice@example.com"}
	rt := reflect.TypeOf(u)
	rv := reflect.ValueOf(u)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		value := rv.Field(i)
		fmt.Printf("  Field: %-8s Type: %-8v Value: %-20v json:%q validate:%q\n",
			field.Name, field.Type, value,
			field.Tag.Get("json"), field.Tag.Get("validate"))
	}

	// ── Setting values via reflection ─────────────────────────────────────
	fmt.Println("\n── Setting values via reflection ──")
	// Must use pointer to be able to set:
	up := &User{}
	rv2 := reflect.ValueOf(up).Elem()
	rv2.FieldByName("Name").SetString("Bob")
	rv2.FieldByName("Age").SetInt(25)
	rv2.FieldByName("Email").SetString("bob@example.com")
	fmt.Printf("  Set via reflection: %+v\n", *up)

	// ── Struct tags (how libraries work) ──────────────────────────────────
	fmt.Println("\n── Struct tags → simpleValidator ──")
	invalid := User{Name: "", Age: 200}
	errs := simpleValidator(invalid)
	for _, e := range errs {
		fmt.Printf("  validation: %s\n", e)
	}
	valid := User{Name: "Alice", Age: 30}
	errs2 := simpleValidator(valid)
	fmt.Printf("  valid user errors: %v\n", errs2)

	// ── structToMap (how JSON encoder works at high level) ─────────────────
	fmt.Println("\n── structToMap ──")
	m2 := structToMap(User{Name: "Carol", Age: 25, Email: "carol@test.com"})
	fmt.Printf("  %v\n", m2)

	// ── reflect.DeepEqual ─────────────────────────────────────────────────
	fmt.Println("\n── reflect.DeepEqual ──")
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	fmt.Printf("  a == b (slices): COMPILE ERROR — can't use ==\n")
	fmt.Printf("  reflect.DeepEqual(a, b): %v\n", reflect.DeepEqual(a, b))
	fmt.Printf("  reflect.DeepEqual([1,2,3], [1,2,4]): %v\n", reflect.DeepEqual(a, []int{1, 2, 4}))
	_ = strconv.Itoa(0) // keep import

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  reflect.TypeOf(v) → Type  (User, int, []string)")
	fmt.Println("  reflect.ValueOf(v) → Value (to read/set)")
	fmt.Println("  Kind = category (struct, ptr, slice, int, ...)")
	fmt.Println("  Must use pointer + .Elem() to set struct fields")
	fmt.Println("  Struct tags: field.Tag.Get(\"json\") — how libs work")
	fmt.Println("  reflect.DeepEqual — compare slices, maps, structs")
	fmt.Println("  Reflection is SLOW — cache TypeOf/ValueOf results, avoid in hot paths")
}

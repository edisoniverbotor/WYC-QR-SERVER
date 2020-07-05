package ReadConfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/structtag"
	"github.com/pschlump/MiscLib"
	"github.com/pschlump/godebug"
	"github.com/pschlump/jsonSyntaxErrorLib"
)

// ReadFile will read a configuration file into the global configuration structure.
func ReadFile(filename string, lCfg interface{}) (err error) {

	// Get the type and value of the argument we were passed.
	ptyp := reflect.TypeOf(lCfg)
	pval := reflect.ValueOf(lCfg)

	// Requries that lCfg is a pointer.
	if ptyp.Kind() != reflect.Ptr {
		fmt.Fprintf(os.Stderr, "Must pass a address of a struct to ReadFile\n")
		os.Exit(1)
	}

	var typ reflect.Type
	var val reflect.Value
	typ = ptyp.Elem()
	val = pval.Elem()

	// Create Defaults

	// Make sure we now have a struct
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("ReadFile was not passed a struct.\n")
	}

	// Can we set values?
	if val.CanSet() {
		if db1 {
			fmt.Printf("Debug: We can set values.\n")
		}
	} else {
		return fmt.Errorf("ReadFile passed a struct that will not allow setting of values\n")
	}

	// The number of fields in the struct is determined by the type of struct
	// it is. Loop through them.
	for i := 0; i < typ.NumField(); i++ {

		// Get the type of the field from the type of the struct. For a struct, you always get a StructField.
		sfld := typ.Field(i)

		// Get the type of the StructField, which is the type actually stored in that field of the struct.
		tfld := sfld.Type

		// Get the Kind of that type, which will be the underlying base type
		// used to define the type in question.
		kind := tfld.Kind()

		// Get the value of the field from the value of the struct.
		vfld := val.Field(i)
		tag := string(sfld.Tag)

		// ... and start using structtag by parsing the tag
		tags, err := structtag.Parse(tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse structure tag ->%s<- %s\n", tag, err)
			os.Exit(1)
		}

		// Dump out what we've found
		if db1 {
			fmt.Printf("Debug: struct field %d: name %s type %s kind %s value %v tag ->%s<- AT:%s\n", i, sfld.Name, tfld, kind, vfld, tag, godebug.LF())

			// iterate over all tags
			for tn, t := range tags.Tags() {
				fmt.Printf("\t[%d] tag: %+v\n", tn, t)
			}

			// get a single tag
			defaultTag, err := tags.Get("default")
			if err != nil {
				fmt.Printf("`default` Not Set\n")
			} else {
				fmt.Println(defaultTag)         // Output: default:"foo,omitempty,string"
				fmt.Println(defaultTag.Key)     // Output: default
				fmt.Println(defaultTag.Name)    // Output: foo
				fmt.Println(defaultTag.Options) // Output: [omitempty string]
			}
		}

		defaultTag, err := tags.Get("default")
		// Is that field some kind of string, and is the value one we can set?
		if kind == reflect.String && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			if err != nil || defaultTag.Name == "" {
				// Ignore error - indicates no "default" tag set.
			} else {
				defaultValue := defaultTag.Name
				if db1 {
					fmt.Printf("Debug: Looking to set field %s to a default value of ->%s<-\n", sfld.Name, defaultValue)
				}
				vfld.SetString(defaultValue)
			}
		} else if kind == reflect.Int && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			if err != nil || defaultTag.Name == "" {
				// Ignore error - indicates no "default" tag set.
			} else {
				defaultValueStr := defaultTag.Name
				defaultValue, err := strconv.ParseInt(defaultValueStr, 10, 64)
				if err != nil {
					return fmt.Errorf("Attempt to set default int value, invalid int ->%s<-, error [%s]", defaultValueStr, err)
				}
				if db1 {
					fmt.Printf("Debug: Looking to set field %s to a default value of ->%v<-\n", sfld.Name, defaultValue)
				}
				vfld.SetInt(defaultValue)
			}
		} else if kind == reflect.Bool && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			if err != nil || defaultTag.Name == "" {
				// Ignore error - indicates no "default" tag set.
			} else {
				defaultValueStr := defaultTag.Name
				defaultValue, err := strconv.ParseBool(defaultValueStr)
				if err != nil {
					return fmt.Errorf("Attempt to set default int value, invalid int ->%s<-, error [%s]", defaultValueStr, err)
				}
				if db1 {
					fmt.Printf("Debug: Looking to set field %s to a default value of ->%v<-\n", sfld.Name, defaultValue)
				}
				vfld.SetBool(defaultValue)
			}
		} else if kind == reflect.Struct && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			recursiveChildStruct(vfld.Addr().Interface())
		} else if kind == reflect.Struct {
			if db3 {
				fmt.Printf("%sProbably an error - can not set - AT: %s%s\n", MiscLib.ColorRed, godebug.LF(), MiscLib.ColorReset)
				panic("oopsy")
			}
		} else if kind != reflect.String && err == nil {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			// report errors - defauilt is only implemented with strings.
			fmt.Fprintf(os.Stderr, "default tag on struct is only implemented for String fields in struct.  Fatal error on %s tag %s\n", sfld.Name, tag)
			os.Exit(1)
		}
	}

	// look for filename in ~/local (C:\local on Winderz)
	var home string
	if os.PathSeparator == '/' {
		home = os.Getenv("HOME")
	} else {
		home = "C:\\"
	}
	homeLocal := path.Join(home, "local")
	base := path.Base(filename)
	if ExistsIsDir(homeLocal) && Exists(path.Join(homeLocal, base)) {
		filename = path.Join(homeLocal, base)
	}
	if db1 {
		fmt.Printf("Debug: File name after checing ~/local [%s]\n", filename)
	}

	var buf []byte
	buf, err = ioutil.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read the JSON file [%s]: error %s\n", filename, err)
		os.Exit(1)
	}

	// err = json.Unmarshal(buf, &gCfg)
	err = json.Unmarshal(buf, lCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid initialization - Unable to parse JSON file, %s\n", err)
		PrintErrorJson(string(buf), err) // show line for error
		os.Exit(1)
	}

	// err = SetFromEnv(&gCfg)
	// err = SetFromEnv(lCfg)
	err = SetFromEnv2(typ, val)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error pulling from environment: %s\n", err)
		os.Exit(1)
	}

	return err
}

func PrintErrorJson(js string, err error) (rv string) {
	rv = jsonSyntaxErrorLib.GenerateSyntaxError(js, err)
	fmt.Fprintf(os.Stderr, "%s\n", rv)
	return
}

func SetFromEnv2(typ reflect.Type, val reflect.Value) (err error) {

	// The number of fields in the struct is determined by the type of struct
	// it is. Loop through them.
	for i := 0; i < typ.NumField(); i++ {

		// Get the type of the field from the type of the struct. For a struct, you always get a StructField.
		sfld := typ.Field(i)

		// Get the type of the StructField, which is the type actually stored in that field of the struct.
		tfld := sfld.Type

		// Get the Kind of that type, which will be the underlying base type
		// used to define the type in question.
		kind := tfld.Kind()

		// Get the value of the field from the value of the struct.
		vfld := val.Field(i)

		// Dump out what we've found
		if db2 {
			fmt.Printf("Debug: struct field %d: name %s type %s kind %s value %v\n", i, sfld.Name, tfld, kind, vfld)
		}

		// Is that field some kind of string, and is the value one we can set?
		if kind == reflect.String && vfld.CanSet() {
			if db2 {
				fmt.Printf("Debug: Looking to set field %s\n", sfld.Name)
			}
			// Assign to it
			curVal := fmt.Sprintf("%s", vfld)
			if len(curVal) > 5 && curVal[0:5] == "$ENV$" {
				envVal := os.Getenv(curVal[5:])
				if db2 {
					fmt.Printf("Debug: %sOverwriting field %s current [%s] with [%s]%s\n", MiscLib.ColorYellow, sfld.Name, curVal, envVal, MiscLib.ColorReset)
				}
				vfld.SetString(envVal)
			}
			if len(curVal) > 6 && curVal[0:6] == "$FILE$" {
				data, err := ioutil.ReadFile(curVal[6:])
				if db2 {
					fmt.Printf("Debug: Overwriting field %s current [%s] with [%s]\n", sfld.Name, data, data)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error [%s] with file [%s] field name [%s]\n", err, curVal[6:], sfld.Name)
					os.Exit(1)
				}
				vfld.SetString(string(data))
			}
		} else if kind == reflect.Struct && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			recursiveSetFromEnv(vfld.Addr().Interface())
		}
	}

	return nil
}

func recursiveSetFromEnv(s interface{}) (err error) {

	// Get the type and value of the argument we were passed.
	ptyp := reflect.TypeOf(s)
	pval := reflect.ValueOf(s)
	// We can't do much with the Value (it's opaque), but we need it in order
	// to fetch individual fields from the struct later.

	var typ reflect.Type
	var val reflect.Value

	// If we were passed a pointer, dereference to get the type and value
	// pointed at.
	if ptyp.Kind() == reflect.Ptr {
		if db2 {
			fmt.Printf("Debug: Argument is a pointer, dereferencing.\n")
		}
		typ = ptyp.Elem()
		val = pval.Elem()
	} else {
		if db2 {
			fmt.Printf("Debug: Argument is %s.%s, a %s.\n", ptyp.PkgPath(), ptyp.Name(), ptyp.Kind())
		}
		typ = ptyp
		val = pval
	}

	// Make sure we now have a struct
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("SetFromEnv was not passed a struct.\n")
	}

	// Can we set values?
	if val.CanSet() {
		if db2 {
			fmt.Printf("Debug: We can set values.\n")
		}
	} else {
		return fmt.Errorf("SetFromEnv passed a struct that will not allow setting of values\n")
	}

	// The number of fields in the struct is determined by the type of struct
	// it is. Loop through them.
	for i := 0; i < typ.NumField(); i++ {

		// Get the type of the field from the type of the struct. For a struct, you always get a StructField.
		sfld := typ.Field(i)

		// Get the type of the StructField, which is the type actually stored in that field of the struct.
		tfld := sfld.Type

		// Get the Kind of that type, which will be the underlying base type
		// used to define the type in question.
		kind := tfld.Kind()

		// Get the value of the field from the value of the struct.
		vfld := val.Field(i)

		// Dump out what we've found
		if db2 {
			fmt.Printf("Debug: struct field %d: name %s type %s kind %s value %v\n", i, sfld.Name, tfld, kind, vfld)
		}

		// Is that field some kind of string, and is the value one we can set?
		if kind == reflect.String && vfld.CanSet() {
			if db2 {
				fmt.Printf("Debug: Looking to set field %s\n", sfld.Name)
			}
			// Assign to it
			curVal := fmt.Sprintf("%s", vfld)
			if len(curVal) > 5 && curVal[0:5] == "$ENV$" {
				envVal := os.Getenv(curVal[5:])
				if db2 {
					fmt.Printf("Debug: Overwriting field %s current [%s] with [%s]\n", sfld.Name, curVal, envVal)
				}
				if len(envVal) > 1 && envVal[0:1] == "~" {
					envVal = ProcessHome(envVal)
				}
				vfld.SetString(envVal)
			}
			if len(curVal) > 6 && curVal[0:6] == "$FILE$" {
				// data, err := ioutil.ReadFile(curVal[6:])
				fn := curVal[6:]
				if len(fn) > 1 && fn[0:1] == "~" {
					fn = ProcessHome(fn)
				}
				data, err := ioutil.ReadFile(fn)
				if db2 {
					fmt.Printf("Debug: Overwriting field %s current [%s] with [%s]\n", sfld.Name, data, data)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error [%s] with file [%s] field name [%s]\n", err, curVal[6:], sfld.Name)
					os.Exit(1)
				}
				vfld.SetString(string(data))
			}
		} else if kind == reflect.Struct && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			recursiveSetFromEnv(vfld.Addr().Interface())
		}
	}

	return nil
}

func SetFromEnv(s interface{}) (err error) {

	// Get the type and value of the argument we were passed.
	ptyp := reflect.TypeOf(s)
	pval := reflect.ValueOf(s)
	// We can't do much with the Value (it's opaque), but we need it in order
	// to fetch individual fields from the struct later.

	var typ reflect.Type
	var val reflect.Value

	// If we were passed a pointer, dereference to get the type and value
	// pointed at.
	if ptyp.Kind() == reflect.Ptr {
		if db2 {
			fmt.Printf("Debug: Argument is a pointer, dereferencing.\n")
		}
		typ = ptyp.Elem()
		val = pval.Elem()
	} else {
		if db2 {
			fmt.Printf("Debug: Argument is %s.%s, a %s.\n", ptyp.PkgPath(), ptyp.Name(), ptyp.Kind())
		}
		typ = ptyp
		val = pval
	}

	// Make sure we now have a struct
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("SetFromEnv was not passed a struct.\n")
	}

	// Can we set values?
	if val.CanSet() {
		if db2 {
			fmt.Printf("Debug: We can set values.\n")
		}
	} else {
		return fmt.Errorf("SetFromEnv passed a struct that will not allow setting of values\n")
	}

	// The number of fields in the struct is determined by the type of struct
	// it is. Loop through them.
	for i := 0; i < typ.NumField(); i++ {

		// Get the type of the field from the type of the struct. For a struct, you always get a StructField.
		sfld := typ.Field(i)

		// Get the type of the StructField, which is the type actually stored in that field of the struct.
		tfld := sfld.Type

		// Get the Kind of that type, which will be the underlying base type
		// used to define the type in question.
		kind := tfld.Kind()

		// Get the value of the field from the value of the struct.
		vfld := val.Field(i)

		// Dump out what we've found
		if db2 {
			fmt.Printf("Debug: struct field %d: name %s type %s kind %s value %v\n", i, sfld.Name, tfld, kind, vfld)
		}

		// Is that field some kind of string, and is the value one we can set?
		if kind == reflect.String && vfld.CanSet() {
			if db2 {
				fmt.Printf("Debug: Looking to set field %s\n", sfld.Name)
			}
			// Assign to it
			curVal := fmt.Sprintf("%s", vfld)
			if len(curVal) > 5 && curVal[0:5] == "$ENV$" {
				envVal := os.Getenv(curVal[5:])
				if db2 {
					fmt.Printf("Debug: Overwriting field %s current [%s] with [%s]\n", sfld.Name, curVal, envVal)
				}
				if len(envVal) > 1 && envVal[0:1] == "~" {
					envVal = ProcessHome(envVal)
				}
				vfld.SetString(envVal)
			}
			if len(curVal) > 6 && curVal[0:6] == "$FILE$" {
				fn := curVal[6:]
				if len(fn) > 1 && fn[0:1] == "~" {
					fn = ProcessHome(fn)
				}
				data, err := ioutil.ReadFile(fn)
				if db2 {
					fmt.Printf("Debug: Overwriting field %s current [%s] with [%s]\n", sfld.Name, data, data)
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error [%s] with file [%s] field name [%s]\n", err, curVal[6:], sfld.Name)
					os.Exit(1)
				}
				vfld.SetString(string(data))
			}
		} else if kind == reflect.Struct && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorYellow, godebug.LF(), MiscLib.ColorReset)
			}
			recursiveSetFromEnv(vfld.Addr().Interface())
		}
	}

	return nil
}

// Exists returns true if a directory or file exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// ExistsIsDir returns true if a direcotry exists.
func ExistsIsDir(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	if fi.IsDir() {
		return true
	}
	return false
}

func recursiveChildStruct(lCfg interface{}) error {

	if db3 {
		fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
	}
	// Get the type and value of the argument we were passed.
	ptyp := reflect.TypeOf(lCfg)
	pval := reflect.ValueOf(lCfg)

	// Requries that lCfg is a pointer.
	if ptyp.Kind() != reflect.Ptr {
		fmt.Fprintf(os.Stderr, "Must pass a address of a struct to ReadFile\n")
		os.Exit(1)
	}

	var typ reflect.Type
	var val reflect.Value
	typ = ptyp.Elem()
	val = pval.Elem()

	// Create Defaults

	// Make sure we now have a struct
	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("ReadFile was not passed a struct.\n")
	}

	// Can we set values?
	if val.CanSet() {
		if db1 {
			fmt.Printf("Debug: We can set values.\n")
		}
	} else {
		return fmt.Errorf("ReadFile passed a struct that will not allow setting of values\n")
	}

	if db3 {
		fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
	}

	// The number of fields in the struct is determined by the type of struct
	// it is. Loop through them.
	for i := 0; i < typ.NumField(); i++ {

		// Get the type of the field from the type of the struct. For a struct, you always get a StructField.
		sfld := typ.Field(i)

		// Get the type of the StructField, which is the type actually stored in that field of the struct.
		tfld := sfld.Type

		// Get the Kind of that type, which will be the underlying base type
		// used to define the type in question.
		kind := tfld.Kind()

		// Get the value of the field from the value of the struct.
		vfld := val.Field(i)
		tag := string(sfld.Tag)

		// ... and start using structtag by parsing the tag
		tags, err := structtag.Parse(tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to parse structure tag ->%s<- %s\n", tag, err)
			os.Exit(1)
		}

		// Dump out what we've found
		if db1 {
			fmt.Printf("Debug: struct field %d: name %s type %s kind %s value %v tag ->%s<- AT:%s\n", i, sfld.Name, tfld, kind, vfld, tag, godebug.LF())

			// iterate over all tags
			for tn, t := range tags.Tags() {
				fmt.Printf("\t[%d] tag: %+v\n", tn, t)
			}

			// get a single tag
			defaultTag, err := tags.Get("default")
			if err != nil {
				fmt.Printf("`default` Not Set\n")
			} else {
				fmt.Println(defaultTag)         // Output: default:"foo,omitempty,string"
				fmt.Println(defaultTag.Key)     // Output: default
				fmt.Println(defaultTag.Name)    // Output: foo
				fmt.Println(defaultTag.Options) // Output: [omitempty string]
			}
		}

		defaultTag, err := tags.Get("default")
		// Is that field some kind of string, and is the value one we can set?
		if kind == reflect.String && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
			}
			if err != nil || defaultTag.Name == "" {
				// Ignore error - indicates no "default" tag set.
			} else {
				defaultValue := defaultTag.Name
				if db1 {
					fmt.Printf("Debug: Looking to set field %s to a default value of ->%s<-\n", sfld.Name, defaultValue)
				}
				vfld.SetString(defaultValue)
			}
		} else if kind == reflect.Int && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
			}
			if err != nil || defaultTag.Name == "" {
				// Ignore error - indicates no "default" tag set.
			} else {
				defaultValueStr := defaultTag.Name
				defaultValue, err := strconv.ParseInt(defaultValueStr, 10, 64)
				if err != nil {
					return fmt.Errorf("Attempt to set default int value, invalid int ->%s<-, error [%s]", defaultValueStr, err)
				}
				if db1 {
					fmt.Printf("Debug: Looking to set field %s to a default value of ->%v<-\n", sfld.Name, defaultValue)
				}
				vfld.SetInt(defaultValue)
			}
		} else if kind == reflect.Bool && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
			}
			if err != nil || defaultTag.Name == "" {
				// Ignore error - indicates no "default" tag set.
			} else {
				defaultValueStr := defaultTag.Name
				defaultValue, err := strconv.ParseBool(defaultValueStr)
				if err != nil {
					return fmt.Errorf("Attempt to set default int value, invalid int ->%s<-, error [%s]", defaultValueStr, err)
				}
				if db1 {
					fmt.Printf("Debug: Looking to set field %s to a default value of ->%v<-\n", sfld.Name, defaultValue)
				}
				vfld.SetBool(defaultValue)
			}
		} else if kind == reflect.Struct && vfld.CanSet() {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
			}
			recursiveChildStruct(vfld.Addr().Interface())
		} else if kind == reflect.Struct {
			if db3 {
				fmt.Printf("%sProbably an error - can not set - AT: %s%s\n", MiscLib.ColorRed, godebug.LF(), MiscLib.ColorReset)
				panic("recursive-oopsy")
			}
		} else if kind != reflect.String && err == nil {
			if db3 {
				fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
			}
			// report errors - defauilt is only implemented with strings.
			fmt.Fprintf(os.Stderr, "default tag on struct is only implemented for String fields in struct.  Fatal error on %s tag %s\n", sfld.Name, tag)
			os.Exit(1)
		}
	}
	if db3 {
		fmt.Printf("%sAT: %s%s\n", MiscLib.ColorCyan, godebug.LF(), MiscLib.ColorReset)
	}
	return nil
}

var home string

func init() {
	if os.PathSeparator == '\\' {
		home = "C:/"
	} else {
		home = os.Getenv("HOME")
	}
}

func ProcessHome(fn string) (outFn string) {
	outFn = fn
	if len(fn) > 1 && fn[0:1] == "~" {
		if len(fn) > 2 && fn[0:2] == "~/" {
			outFn = path.Join(home, fn[2:])
			return
		} else {
			s1 := strings.Split(fn[1:], "/")
			username := s1[0]
			uu, err := user.Lookup(username)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to lookup [%s] user and get home directory.\n", username)
				return
			}
			outFn = path.Join(uu.HomeDir, strings.Join(s1[1:], "/"))
			return
		}
	}
	return
}

var db1 = false
var db2 = false
var db3 = false
var db4 = false

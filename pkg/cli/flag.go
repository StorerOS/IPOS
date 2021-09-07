package cli

import (
	"flag"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const defaultPlaceholder = "value"

var BashCompletionFlag Flag = BoolFlag{
	Name:   "generate-bash-completion",
	Hidden: true,
}

var VersionFlag Flag = BoolFlag{
	Name:  "version, v",
	Usage: "print the version",
}

var HelpFlag Flag = BoolFlag{
	Name:  "help, h",
	Usage: "show help",
}

var FlagStringer FlagStringFunc = stringifyFlag

type FlagsByName []Flag

func (f FlagsByName) Len() int {
	return len(f)
}

func (f FlagsByName) Less(i, j int) bool {
	return f[i].GetName() < f[j].GetName()
}

func (f FlagsByName) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type Flag interface {
	fmt.Stringer
	Apply(*flag.FlagSet)
	GetName() string
}

type errorableFlag interface {
	Flag

	ApplyWithError(*flag.FlagSet) error
}

func flagSet(name string, flags []Flag) (*flag.FlagSet, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)

	for _, f := range flags {
		if ef, ok := f.(errorableFlag); ok {
			if err := ef.ApplyWithError(set); err != nil {
				return nil, err
			}
		} else {
			f.Apply(set)
		}
	}
	return set, nil
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

type Generic interface {
	Set(value string) error
	String() string
}

func (f GenericFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f GenericFlag) ApplyWithError(set *flag.FlagSet) error {
	val := f.Value
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				if err := val.Set(envVal); err != nil {
					return fmt.Errorf("could not parse %s as value for flag %s: %s", envVal, f.Name, err)
				}
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		set.Var(f.Value, name, f.Usage)
	})

	return nil
}

type StringSlice []string

func (f *StringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (f *StringSlice) String() string {
	return fmt.Sprintf("%s", *f)
}

func (f *StringSlice) Value() []string {
	return *f
}

func (f *StringSlice) Get() interface{} {
	return *f
}

func (f StringSliceFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f StringSliceFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				newVal := &StringSlice{}
				for _, s := range strings.Split(envVal, ",") {
					s = strings.TrimSpace(s)
					if err := newVal.Set(s); err != nil {
						return fmt.Errorf("could not parse %s as string value for flag %s: %s", envVal, f.Name, err)
					}
				}
				f.Value = newVal
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Value == nil {
			f.Value = &StringSlice{}
		}
		set.Var(f.Value, name, f.Usage)
	})

	return nil
}

type IntSlice []int

func (f *IntSlice) Set(value string) error {
	tmp, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*f = append(*f, tmp)
	return nil
}

func (f *IntSlice) String() string {
	return fmt.Sprintf("%#v", *f)
}

func (f *IntSlice) Value() []int {
	return *f
}

func (f *IntSlice) Get() interface{} {
	return *f
}

func (f IntSliceFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f IntSliceFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				newVal := &IntSlice{}
				for _, s := range strings.Split(envVal, ",") {
					s = strings.TrimSpace(s)
					if err := newVal.Set(s); err != nil {
						return fmt.Errorf("could not parse %s as int slice value for flag %s: %s", envVal, f.Name, err)
					}
				}
				f.Value = newVal
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Value == nil {
			f.Value = &IntSlice{}
		}
		set.Var(f.Value, name, f.Usage)
	})

	return nil
}

type Int64Slice []int64

func (f *Int64Slice) Set(value string) error {
	tmp, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	*f = append(*f, tmp)
	return nil
}

func (f *Int64Slice) String() string {
	return fmt.Sprintf("%#v", *f)
}

func (f *Int64Slice) Value() []int64 {
	return *f
}

func (f *Int64Slice) Get() interface{} {
	return *f
}

func (f Int64SliceFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f Int64SliceFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				newVal := &Int64Slice{}
				for _, s := range strings.Split(envVal, ",") {
					s = strings.TrimSpace(s)
					if err := newVal.Set(s); err != nil {
						return fmt.Errorf("could not parse %s as int64 slice value for flag %s: %s", envVal, f.Name, err)
					}
				}
				f.Value = newVal
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Value == nil {
			f.Value = &Int64Slice{}
		}
		set.Var(f.Value, name, f.Usage)
	})
	return nil
}

func (f BoolFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f BoolFlag) ApplyWithError(set *flag.FlagSet) error {
	val := false
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				if envVal == "" {
					val = false
					break
				}

				envValBool, err := strconv.ParseBool(envVal)
				if err != nil {
					return fmt.Errorf("could not parse %s as bool value for flag %s: %s", envVal, f.Name, err)
				}

				val = envValBool
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.BoolVar(f.Destination, name, val, f.Usage)
			return
		}
		set.Bool(name, val, f.Usage)
	})

	return nil
}

func (f BoolTFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f BoolTFlag) ApplyWithError(set *flag.FlagSet) error {
	val := true
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				if envVal == "" {
					val = false
					break
				}

				envValBool, err := strconv.ParseBool(envVal)
				if err != nil {
					return fmt.Errorf("could not parse %s as bool value for flag %s: %s", envVal, f.Name, err)
				}

				val = envValBool
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.BoolVar(f.Destination, name, val, f.Usage)
			return
		}
		set.Bool(name, val, f.Usage)
	})

	return nil
}

func (f StringFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f StringFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				f.Value = envVal
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.StringVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.String(name, f.Value, f.Usage)
	})

	return nil
}

func (f IntFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f IntFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				envValInt, err := strconv.ParseInt(envVal, 0, 64)
				if err != nil {
					return fmt.Errorf("could not parse %s as int value for flag %s: %s", envVal, f.Name, err)
				}
				f.Value = int(envValInt)
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.IntVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Int(name, f.Value, f.Usage)
	})

	return nil
}

func (f Int64Flag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f Int64Flag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				envValInt, err := strconv.ParseInt(envVal, 0, 64)
				if err != nil {
					return fmt.Errorf("could not parse %s as int value for flag %s: %s", envVal, f.Name, err)
				}

				f.Value = envValInt
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.Int64Var(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Int64(name, f.Value, f.Usage)
	})

	return nil
}

func (f UintFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f UintFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				envValInt, err := strconv.ParseUint(envVal, 0, 64)
				if err != nil {
					return fmt.Errorf("could not parse %s as uint value for flag %s: %s", envVal, f.Name, err)
				}

				f.Value = uint(envValInt)
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.UintVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Uint(name, f.Value, f.Usage)
	})

	return nil
}

func (f Uint64Flag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f Uint64Flag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				envValInt, err := strconv.ParseUint(envVal, 0, 64)
				if err != nil {
					return fmt.Errorf("could not parse %s as uint64 value for flag %s: %s", envVal, f.Name, err)
				}

				f.Value = uint64(envValInt)
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.Uint64Var(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Uint64(name, f.Value, f.Usage)
	})

	return nil
}

func (f DurationFlag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f DurationFlag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				envValDuration, err := time.ParseDuration(envVal)
				if err != nil {
					return fmt.Errorf("could not parse %s as duration for flag %s: %s", envVal, f.Name, err)
				}

				f.Value = envValDuration
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.DurationVar(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Duration(name, f.Value, f.Usage)
	})

	return nil
}

func (f Float64Flag) Apply(set *flag.FlagSet) {
	f.ApplyWithError(set)
}

func (f Float64Flag) ApplyWithError(set *flag.FlagSet) error {
	if f.EnvVar != "" {
		for _, envVar := range strings.Split(f.EnvVar, ",") {
			envVar = strings.TrimSpace(envVar)
			if envVal, ok := syscall.Getenv(envVar); ok {
				envValFloat, err := strconv.ParseFloat(envVal, 10)
				if err != nil {
					return fmt.Errorf("could not parse %s as float64 value for flag %s: %s", envVal, f.Name, err)
				}

				f.Value = float64(envValFloat)
				break
			}
		}
	}

	eachName(f.Name, func(name string) {
		if f.Destination != nil {
			set.Float64Var(f.Destination, name, f.Value, f.Usage)
			return
		}
		set.Float64(name, f.Value, f.Usage)
	})

	return nil
}

func visibleFlags(fl []Flag) []Flag {
	visible := []Flag{}
	for _, flag := range fl {
		field := flagValue(flag).FieldByName("Hidden")
		if !field.IsValid() || !field.Bool() {
			visible = append(visible, flag)
		}
	}
	return visible
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	return
}

func unquoteUsage(usage string) (string, string) {
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name := usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			break
		}
	}
	return "", usage
}

func prefixedNames(fullName, placeholder string) string {
	var prefixed string
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if placeholder != "" {
			prefixed += " " + placeholder
		}
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return prefixed
}

func withEnvHint(envVar, str string) string {
	envText := ""
	if envVar != "" {
		prefix := "$"
		suffix := ""
		sep := ", $"
		if runtime.GOOS == "windows" {
			prefix = "%"
			suffix = "%"
			sep = "%, %"
		}
		envText = fmt.Sprintf(" [%s%s%s]", prefix, strings.Join(strings.Split(envVar, ","), sep), suffix)
	}
	return str + envText
}

func flagValue(f Flag) reflect.Value {
	fv := reflect.ValueOf(f)
	for fv.Kind() == reflect.Ptr {
		fv = reflect.Indirect(fv)
	}
	return fv
}

func stringifyFlag(f Flag) string {
	fv := flagValue(f)

	switch f.(type) {
	case IntSliceFlag:
		return withEnvHint(fv.FieldByName("EnvVar").String(),
			stringifyIntSliceFlag(f.(IntSliceFlag)))
	case Int64SliceFlag:
		return withEnvHint(fv.FieldByName("EnvVar").String(),
			stringifyInt64SliceFlag(f.(Int64SliceFlag)))
	case StringSliceFlag:
		return withEnvHint(fv.FieldByName("EnvVar").String(),
			stringifyStringSliceFlag(f.(StringSliceFlag)))
	}

	placeholder, usage := unquoteUsage(fv.FieldByName("Usage").String())

	needsPlaceholder := false
	defaultValueString := ""

	if val := fv.FieldByName("Value"); val.IsValid() {
		needsPlaceholder = true
		defaultValueString = fmt.Sprintf(" (default: %v)", val.Interface())

		if val.Kind() == reflect.String && val.String() != "" {
			defaultValueString = fmt.Sprintf(" (default: %q)", val.String())
		}
	}

	if defaultValueString == " (default: )" {
		defaultValueString = ""
	}

	if needsPlaceholder && placeholder == "" {
		placeholder = defaultPlaceholder
	}

	usageWithDefault := strings.TrimSpace(fmt.Sprintf("%s%s", usage, defaultValueString))

	return withEnvHint(fv.FieldByName("EnvVar").String(),
		fmt.Sprintf("%s\t%s", prefixedNames(fv.FieldByName("Name").String(), placeholder), usageWithDefault))
}

func stringifyIntSliceFlag(f IntSliceFlag) string {
	defaultVals := []string{}
	if f.Value != nil && len(f.Value.Value()) > 0 {
		for _, i := range f.Value.Value() {
			defaultVals = append(defaultVals, fmt.Sprintf("%d", i))
		}
	}

	return stringifySliceFlag(f.Usage, f.Name, defaultVals)
}

func stringifyInt64SliceFlag(f Int64SliceFlag) string {
	defaultVals := []string{}
	if f.Value != nil && len(f.Value.Value()) > 0 {
		for _, i := range f.Value.Value() {
			defaultVals = append(defaultVals, fmt.Sprintf("%d", i))
		}
	}

	return stringifySliceFlag(f.Usage, f.Name, defaultVals)
}

func stringifyStringSliceFlag(f StringSliceFlag) string {
	defaultVals := []string{}
	if f.Value != nil && len(f.Value.Value()) > 0 {
		for _, s := range f.Value.Value() {
			if len(s) > 0 {
				defaultVals = append(defaultVals, fmt.Sprintf("%q", s))
			}
		}
	}

	return stringifySliceFlag(f.Usage, f.Name, defaultVals)
}

func stringifySliceFlag(usage, name string, defaultVals []string) string {
	placeholder, usage := unquoteUsage(usage)
	if placeholder == "" {
		placeholder = defaultPlaceholder
	}

	defaultVal := ""
	if len(defaultVals) > 0 {
		defaultVal = fmt.Sprintf(" (default: %s)", strings.Join(defaultVals, ", "))
	}

	usageWithDefault := strings.TrimSpace(fmt.Sprintf("%s%s", usage, defaultVal))
	return fmt.Sprintf("%s\t%s", prefixedNames(name, placeholder), usageWithDefault)
}
package altsrc

import (
	"flag"

	"gopkg.in/urfave/cli.v1"
)

type BoolFlag struct {
	cli.BoolFlag
	set *flag.FlagSet
}

func NewBoolFlag(fl cli.BoolFlag) *BoolFlag {
	return &BoolFlag{BoolFlag: fl, set: nil}
}

func (f *BoolFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.BoolFlag.Apply(set)
}

func (f *BoolFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.BoolFlag.ApplyWithError(set)
}

type BoolTFlag struct {
	cli.BoolTFlag
	set *flag.FlagSet
}

func NewBoolTFlag(fl cli.BoolTFlag) *BoolTFlag {
	return &BoolTFlag{BoolTFlag: fl, set: nil}
}

func (f *BoolTFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.BoolTFlag.Apply(set)
}

func (f *BoolTFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.BoolTFlag.ApplyWithError(set)
}

type DurationFlag struct {
	cli.DurationFlag
	set *flag.FlagSet
}

func NewDurationFlag(fl cli.DurationFlag) *DurationFlag {
	return &DurationFlag{DurationFlag: fl, set: nil}
}

func (f *DurationFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.DurationFlag.Apply(set)
}

func (f *DurationFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.DurationFlag.ApplyWithError(set)
}

type Float64Flag struct {
	cli.Float64Flag
	set *flag.FlagSet
}

func NewFloat64Flag(fl cli.Float64Flag) *Float64Flag {
	return &Float64Flag{Float64Flag: fl, set: nil}
}

func (f *Float64Flag) Apply(set *flag.FlagSet) {
	f.set = set
	f.Float64Flag.Apply(set)
}

func (f *Float64Flag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.Float64Flag.ApplyWithError(set)
}

type GenericFlag struct {
	cli.GenericFlag
	set *flag.FlagSet
}

func NewGenericFlag(fl cli.GenericFlag) *GenericFlag {
	return &GenericFlag{GenericFlag: fl, set: nil}
}

func (f *GenericFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.GenericFlag.Apply(set)
}

func (f *GenericFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.GenericFlag.ApplyWithError(set)
}

type Int64Flag struct {
	cli.Int64Flag
	set *flag.FlagSet
}

func NewInt64Flag(fl cli.Int64Flag) *Int64Flag {
	return &Int64Flag{Int64Flag: fl, set: nil}
}

func (f *Int64Flag) Apply(set *flag.FlagSet) {
	f.set = set
	f.Int64Flag.Apply(set)
}

func (f *Int64Flag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.Int64Flag.ApplyWithError(set)
}

type IntFlag struct {
	cli.IntFlag
	set *flag.FlagSet
}

func NewIntFlag(fl cli.IntFlag) *IntFlag {
	return &IntFlag{IntFlag: fl, set: nil}
}

func (f *IntFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.IntFlag.Apply(set)
}

func (f *IntFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.IntFlag.ApplyWithError(set)
}

type IntSliceFlag struct {
	cli.IntSliceFlag
	set *flag.FlagSet
}

func NewIntSliceFlag(fl cli.IntSliceFlag) *IntSliceFlag {
	return &IntSliceFlag{IntSliceFlag: fl, set: nil}
}

func (f *IntSliceFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.IntSliceFlag.Apply(set)
}

func (f *IntSliceFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.IntSliceFlag.ApplyWithError(set)
}

type Int64SliceFlag struct {
	cli.Int64SliceFlag
	set *flag.FlagSet
}

func NewInt64SliceFlag(fl cli.Int64SliceFlag) *Int64SliceFlag {
	return &Int64SliceFlag{Int64SliceFlag: fl, set: nil}
}

func (f *Int64SliceFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.Int64SliceFlag.Apply(set)
}

func (f *Int64SliceFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.Int64SliceFlag.ApplyWithError(set)
}

type StringFlag struct {
	cli.StringFlag
	set *flag.FlagSet
}

func NewStringFlag(fl cli.StringFlag) *StringFlag {
	return &StringFlag{StringFlag: fl, set: nil}
}

func (f *StringFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.StringFlag.Apply(set)
}

func (f *StringFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.StringFlag.ApplyWithError(set)
}

type StringSliceFlag struct {
	cli.StringSliceFlag
	set *flag.FlagSet
}

func NewStringSliceFlag(fl cli.StringSliceFlag) *StringSliceFlag {
	return &StringSliceFlag{StringSliceFlag: fl, set: nil}
}

func (f *StringSliceFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.StringSliceFlag.Apply(set)
}

func (f *StringSliceFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.StringSliceFlag.ApplyWithError(set)
}

type Uint64Flag struct {
	cli.Uint64Flag
	set *flag.FlagSet
}

func NewUint64Flag(fl cli.Uint64Flag) *Uint64Flag {
	return &Uint64Flag{Uint64Flag: fl, set: nil}
}

func (f *Uint64Flag) Apply(set *flag.FlagSet) {
	f.set = set
	f.Uint64Flag.Apply(set)
}

func (f *Uint64Flag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.Uint64Flag.ApplyWithError(set)
}

type UintFlag struct {
	cli.UintFlag
	set *flag.FlagSet
}

func NewUintFlag(fl cli.UintFlag) *UintFlag {
	return &UintFlag{UintFlag: fl, set: nil}
}

func (f *UintFlag) Apply(set *flag.FlagSet) {
	f.set = set
	f.UintFlag.Apply(set)
}

func (f *UintFlag) ApplyWithError(set *flag.FlagSet) error {
	f.set = set
	return f.UintFlag.ApplyWithError(set)
}

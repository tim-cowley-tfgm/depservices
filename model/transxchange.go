package model

type TransXChange struct {
	Operators Operators
	Services  Services
}

type Operators struct {
	LicensedOperator []LicensedOperator
	Operator         []Operator
}

type LicensedOperator struct {
	OperatorCode string
}

type Operator struct {
	OperatorCode string
}

type Services struct {
	Service []Service
}

type Service struct {
	Lines           Lines
	Description     string
	OperatingPeriod OperatingPeriod
}

type Lines struct {
	Line []Line
}

type Line struct {
	LineName string
}

type OperatingPeriod struct {
	StartDate string
	EndDate   string
}

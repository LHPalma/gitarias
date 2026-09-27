package config

type Branches struct {
	Protected       []string
	ProtectedSource Source
	Base            string
	BaseSource      Source
}

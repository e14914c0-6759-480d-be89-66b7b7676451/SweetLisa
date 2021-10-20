package model

type Passage struct {
	In  In
	Out *Out `json:",omitempty"`
}

type In struct {
	From string `json:",omitempty"`
	Argument
}

type Out struct {
	To   string
	Host string
	Port string
	Argument
}

package repository

type RedisCommand struct {
	Name   string
	Args   []interface{}
	Result chan RedisResult
}

type RedisResult struct {
	Err   error
	Value interface{}
}

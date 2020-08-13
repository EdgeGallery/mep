module mepauth

go 1.14

replace (
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65 => golang.org/x/net v0.0.0-20200301022130-244492dfa37a
	golang.org/x/text v0.3.2 => golang.org/x/text v0.3.3
)

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/astaxie/beego v1.12.0
	github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/go-playground/validator/v10 v10.2.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/shiena/ansicolor v0.0.0-20151119151921-a422bbe96644 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)
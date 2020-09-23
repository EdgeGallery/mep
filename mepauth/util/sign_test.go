package util

import (
	"net/http"
	"strings"
	"testing"


	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetSignature(t *testing.T) {
	Convey("clear byte array", t, func() {
		s := Sign{
			AccessKey: "00000000",
			SecretKey: []byte("00000000"),
		}
		host := "127.0.0.1:8080"
		reqUrl := "https://" + host + "/mepauth/mepauth/v1/token"
		reqToBeSigned, errNewRequest := http.NewRequest("POST", reqUrl, strings.NewReader(""))
		if errNewRequest != nil {
			log.Error("prepare http request to generate signature is failed")
		}
		reqToBeSigned.Header.Set("content-type", "json")
		reqToBeSigned.Header.Set(HOST_HEADER, host)
		reqToBeSigned.Header.Set(DATE_HEADER, DATE_FORMAT)
		signature, err := s.GetSignature(reqToBeSigned)
		So(signature, ShouldNotBeNil)
		So(err, ShouldBeNil)
	})
}



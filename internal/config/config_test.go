package config

import "testing"

func TestValidateRejectsDuplicateProviders(t *testing.T){c:=Config{Address:":1",Providers:[]ProviderConfig{{Name:"same",Type:"mock"},{Name:"same",Type:"mock"}}};if c.Validate()==nil{t.Fatal("expected duplicate provider error")}}

package httpmock

func CheckErr(e error) {
	if e != nil {
		panic(e)
	}
}
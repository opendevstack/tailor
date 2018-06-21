package openshift

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestNewParamsFromInput(t *testing.T) {

	input := []byte(
		`FOO.ENC=c2VjcmV0
BAR.STRING=geheim
BAZ=hello_world
`)

	actual := NewParamsFromInput(string(input))
	expected := ParamsFromInput{
		&ParamFromInput{
			Key:      "FOO",
			IsSecret: true,
			Value:    "c2VjcmV0",
		},
		&ParamFromInput{
			Key:      "BAR",
			IsSecret: true,
			Value:    "Z2VoZWlt",
		},
		&ParamFromInput{
			Key:   "BAZ",
			Value: "hello_world",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Params don't match, got: %v, want: %v.", actual.String(), expected.String())
	}
}

// func TestNewParamsFromFile(t *testing.T) {

//   content := []byte(
//     `FOO.ENC=wcFMA7dOewnVojzNARAAO5F+fyhjOGLB/60JqbqNB/2+WAcB+ikSrNlxVh0VxlLcPOVgJXrfRAMiTCAE13A9xpT0sg/SCZJotmCdwxm87E/vXfukjZIWz6JLX4A8QXqde3rx3qAwOL/9HA4esLsbQb+yHJBWwiNxuPWs1uSiTuMH1M0Ol6ieL+Ui/QEPWoz72r4PrpgbPQdsrGvk30kviC8srGQAfojHYehLIZlcezzb0mE+ssTpraXGg983QS6ByNlOHJXyW+2ipSe22jsYOLjL41WQ+WZTKcIVFvgANfR+X6jZt5G03TQ6wGZNYBgxQB0M/RY7Eu8XxgmiLqTNZWdJyUDu4LWjakc9NEf6RQsnrgDv8ZXVnl83X2w+GNlgFJwydVpV54Irsllw9t5RUhl3E9QX4RXDOClqxgCnLWJBi2y83UApiTZVAjrZaHa6l7AZaFfjlU91MIqjjUkRsAV76yfE53bFzmku5k7xcYzNB87yI4ZH97ImHRU7rzGdG1pKgC46ho/d9qF2IM7X5wK7AjVogFPT4L4zlw96ori/25OQn9qQ7tZd7GL0ymj1C6K/VPCUclk9PXMKHlbta1T6lct8xRXNWyGlVgf3gqmuAYjiP+D+h4aS39IH5pjlPOc57YC1Wex2R2Aq30i3Vb2gwNJf47BvMDcOMcWfYKy8NnnKJqRbHlhIfW6pHZ7S4AHkYcXe+L5kZIpWGOvYWdoHtOGALuBl4AfhvFXgCuLAa4th4OXjfUkHy7+lq8DgVeQFUFQpdCWdotL/njShHT7s4uVq+x/hTH4A
// BAR.ENC=wcFMA7dOewnVojzNARAAjzKhMvcneu8cMbKDofoP2CEQ5axlmoAwqMndFfnKXU/mMLKbPAcxtvFKm6/pI7nfFGPyftIwkYvda4BT/gR76Gptes5M77e6AodxrATI5ClFKgcMzSp/Mdysf+hlulwAqtAiOdkSYnYOLB2/y1EPs/chWzLwBzwtgiBPJqIrSzCaDY/m2bNjjcePZ/wI/PkoQk+SyrPFmcLOwlvFiHHKdsz0zaIMjW5cgee0fO/Pqr3iQDfzqiWI8QSpXBgdv5Qx4iqLZ7iBClXnqc+DfavnGhV91/rEvUcFSuxmm43VDQy+475xw1Py9fdT4Vs/F8TYxTMPozAOr1vNjuBlMRX0nq1p3leharmmtQkwXmZut53NZMMvhTSqLjKEsM9LHm/OT3nAwsOwE/pah8lOlU2dGg9voIYzhWHyqoEE9yhxmQE3pnLgnN+xTX124QU1Kf72zxoImmAKac78i020tzImGHKCXG1wf5gmajKKE+tHwzRByKqJDUsjbYxvxVBtrWrNAx5ui+gFYrGdFrqP76HdCytHN64tMAiw8fR/R4AiJwg9xHUPYi6dt09vWxajBhWjx0qyI9FbZmFmrCpxW7YnvoBMJV9S9kzpbNicZ2ks2m8yr+HxYTGdSaChSUivorY9R+xz5l7Ei5crZvC6unGCH2ZJpOm5b1tbiX9jcYP0MBrS4AHkw7U6LfYCeUTJEeVxMcUmN+EyA+CI4MjhDDbgcOJUopmF4ATj6RU/sNjk4m3g9uSws9aMoibleLyTpjbBwqln4lg44P3hpc8A
// BAZ=hello_world
// `)

//   defer os.Remove("foo.env")
//   ioutil.WriteFile("foo.env", content, 0644)

//   actual, _ := NewParamsFromFile("foo.env")
//   expected := ParamsFromFile{
//     &ParamFromFile{
//       Key:  "FOO",
//       IsSecret: true,
//       Value: "wcFMA7dOewnVojzNARAAO5F+fyhjOGLB/60JqbqNB/2+WAcB+ikSrNlxVh0VxlLcPOVgJXrfRAMiTCAE13A9xpT0sg/SCZJotmCdwxm87E/vXfukjZIWz6JLX4A8QXqde3rx3qAwOL/9HA4esLsbQb+yHJBWwiNxuPWs1uSiTuMH1M0Ol6ieL+Ui/QEPWoz72r4PrpgbPQdsrGvk30kviC8srGQAfojHYehLIZlcezzb0mE+ssTpraXGg983QS6ByNlOHJXyW+2ipSe22jsYOLjL41WQ+WZTKcIVFvgANfR+X6jZt5G03TQ6wGZNYBgxQB0M/RY7Eu8XxgmiLqTNZWdJyUDu4LWjakc9NEf6RQsnrgDv8ZXVnl83X2w+GNlgFJwydVpV54Irsllw9t5RUhl3E9QX4RXDOClqxgCnLWJBi2y83UApiTZVAjrZaHa6l7AZaFfjlU91MIqjjUkRsAV76yfE53bFzmku5k7xcYzNB87yI4ZH97ImHRU7rzGdG1pKgC46ho/d9qF2IM7X5wK7AjVogFPT4L4zlw96ori/25OQn9qQ7tZd7GL0ymj1C6K/VPCUclk9PXMKHlbta1T6lct8xRXNWyGlVgf3gqmuAYjiP+D+h4aS39IH5pjlPOc57YC1Wex2R2Aq30i3Vb2gwNJf47BvMDcOMcWfYKy8NnnKJqRbHlhIfW6pHZ7S4AHkYcXe+L5kZIpWGOvYWdoHtOGALuBl4AfhvFXgCuLAa4th4OXjfUkHy7+lq8DgVeQFUFQpdCWdotL/njShHT7s4uVq+x/hTH4A",
//     },
//     &ParamFromFile{
//       Key:  "BAR",
//       IsSecret: true,
//       Value: "wcFMA7dOewnVojzNARAAjzKhMvcneu8cMbKDofoP2CEQ5axlmoAwqMndFfnKXU/mMLKbPAcxtvFKm6/pI7nfFGPyftIwkYvda4BT/gR76Gptes5M77e6AodxrATI5ClFKgcMzSp/Mdysf+hlulwAqtAiOdkSYnYOLB2/y1EPs/chWzLwBzwtgiBPJqIrSzCaDY/m2bNjjcePZ/wI/PkoQk+SyrPFmcLOwlvFiHHKdsz0zaIMjW5cgee0fO/Pqr3iQDfzqiWI8QSpXBgdv5Qx4iqLZ7iBClXnqc+DfavnGhV91/rEvUcFSuxmm43VDQy+475xw1Py9fdT4Vs/F8TYxTMPozAOr1vNjuBlMRX0nq1p3leharmmtQkwXmZut53NZMMvhTSqLjKEsM9LHm/OT3nAwsOwE/pah8lOlU2dGg9voIYzhWHyqoEE9yhxmQE3pnLgnN+xTX124QU1Kf72zxoImmAKac78i020tzImGHKCXG1wf5gmajKKE+tHwzRByKqJDUsjbYxvxVBtrWrNAx5ui+gFYrGdFrqP76HdCytHN64tMAiw8fR/R4AiJwg9xHUPYi6dt09vWxajBhWjx0qyI9FbZmFmrCpxW7YnvoBMJV9S9kzpbNicZ2ks2m8yr+HxYTGdSaChSUivorY9R+xz5l7Ei5crZvC6unGCH2ZJpOm5b1tbiX9jcYP0MBrS4AHkw7U6LfYCeUTJEeVxMcUmN+EyA+CI4MjhDDbgcOJUopmF4ATj6RU/sNjk4m3g9uSws9aMoibleLyTpjbBwqln4lg44P3hpc8A",
//     },
//     &ParamFromFile{
//       Key:  "BAZ",
//       Value: "hello_world",
//     },
//   }

//   if !reflect.DeepEqual(actual, expected) {
//     t.Errorf("Params don't match, got: %v, want: %v.", actual.String(), expected.String())
//   }
// }

func TestProcessParamsFromFile(t *testing.T) {

	content := []byte(
		`-----BEGIN PGP PRIVATE KEY BLOCK-----

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
=nDrt
-----END PGP PRIVATE KEY BLOCK-----
`)

	defer os.Remove("private.key")
	ioutil.WriteFile("private.key", content, 0644)

	params := ParamsFromFile{
		&ParamFromFile{
			Key:       "FOO",
			IsSecret:  true,
			Value:     "wcFMA7dOewnVojzNARAAO5F+fyhjOGLB/60JqbqNB/2+WAcB+ikSrNlxVh0VxlLcPOVgJXrfRAMiTCAE13A9xpT0sg/SCZJotmCdwxm87E/vXfukjZIWz6JLX4A8QXqde3rx3qAwOL/9HA4esLsbQb+yHJBWwiNxuPWs1uSiTuMH1M0Ol6ieL+Ui/QEPWoz72r4PrpgbPQdsrGvk30kviC8srGQAfojHYehLIZlcezzb0mE+ssTpraXGg983QS6ByNlOHJXyW+2ipSe22jsYOLjL41WQ+WZTKcIVFvgANfR+X6jZt5G03TQ6wGZNYBgxQB0M/RY7Eu8XxgmiLqTNZWdJyUDu4LWjakc9NEf6RQsnrgDv8ZXVnl83X2w+GNlgFJwydVpV54Irsllw9t5RUhl3E9QX4RXDOClqxgCnLWJBi2y83UApiTZVAjrZaHa6l7AZaFfjlU91MIqjjUkRsAV76yfE53bFzmku5k7xcYzNB87yI4ZH97ImHRU7rzGdG1pKgC46ho/d9qF2IM7X5wK7AjVogFPT4L4zlw96ori/25OQn9qQ7tZd7GL0ymj1C6K/VPCUclk9PXMKHlbta1T6lct8xRXNWyGlVgf3gqmuAYjiP+D+h4aS39IH5pjlPOc57YC1Wex2R2Aq30i3Vb2gwNJf47BvMDcOMcWfYKy8NnnKJqRbHlhIfW6pHZ7S4AHkYcXe+L5kZIpWGOvYWdoHtOGALuBl4AfhvFXgCuLAa4th4OXjfUkHy7+lq8DgVeQFUFQpdCWdotL/njShHT7s4uVq+x/hTH4A",
			Decrypted: "c2VjcmV0",
		},
		&ParamFromFile{
			Key:       "BAR",
			IsSecret:  true,
			Value:     "wcFMA7dOewnVojzNARAAjzKhMvcneu8cMbKDofoP2CEQ5axlmoAwqMndFfnKXU/mMLKbPAcxtvFKm6/pI7nfFGPyftIwkYvda4BT/gR76Gptes5M77e6AodxrATI5ClFKgcMzSp/Mdysf+hlulwAqtAiOdkSYnYOLB2/y1EPs/chWzLwBzwtgiBPJqIrSzCaDY/m2bNjjcePZ/wI/PkoQk+SyrPFmcLOwlvFiHHKdsz0zaIMjW5cgee0fO/Pqr3iQDfzqiWI8QSpXBgdv5Qx4iqLZ7iBClXnqc+DfavnGhV91/rEvUcFSuxmm43VDQy+475xw1Py9fdT4Vs/F8TYxTMPozAOr1vNjuBlMRX0nq1p3leharmmtQkwXmZut53NZMMvhTSqLjKEsM9LHm/OT3nAwsOwE/pah8lOlU2dGg9voIYzhWHyqoEE9yhxmQE3pnLgnN+xTX124QU1Kf72zxoImmAKac78i020tzImGHKCXG1wf5gmajKKE+tHwzRByKqJDUsjbYxvxVBtrWrNAx5ui+gFYrGdFrqP76HdCytHN64tMAiw8fR/R4AiJwg9xHUPYi6dt09vWxajBhWjx0qyI9FbZmFmrCpxW7YnvoBMJV9S9kzpbNicZ2ks2m8yr+HxYTGdSaChSUivorY9R+xz5l7Ei5crZvC6unGCH2ZJpOm5b1tbiX9jcYP0MBrS4AHkw7U6LfYCeUTJEeVxMcUmN+EyA+CI4MjhDDbgcOJUopmF4ATj6RU/sNjk4m3g9uSws9aMoibleLyTpjbBwqln4lg44P3hpc8A",
			Decrypted: "Z2VoZWlt",
		},
		&ParamFromFile{
			Key:   "BAZ",
			Value: "hello_world",
		},
	}

	actual := params.Process(true)
	expected := []byte(
		`FOO=c2VjcmV0
BAR=Z2VoZWlt
BAZ=hello_world
`)

	if !reflect.DeepEqual(actual, string(expected)) {
		t.Errorf("Params don't match, got: %v, want: %v.", actual, string(expected))
	}
}

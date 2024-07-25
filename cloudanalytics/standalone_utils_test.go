package cloudanalytics

import (
	"fmt"
	"testing"
)

func Test_solveFormula(t *testing.T) {
	type args struct {
		formula []string
		vars    map[string]bool
	}

	type test struct {
		name string
		args args
		want bool
	}

	tests := []test{}

	bools := []bool{true, false}

	vars := map[string]bool{"A": true, "B": true, "C": true, "D": true}

	for _, b1 := range bools {
		vars["A"] = b1

		for _, b2 := range bools {
			vars["B"] = b2

			for _, b3 := range bools {
				vars["C"] = b3
				result := vars["A"] && (vars["B"] || vars["C"])

				tests = append(tests, test{
					fmt.Sprintf("test 1 with %v+", vars),
					args{[]string{"A", "AND", "(B", "OR", "C)"},
						map[string]bool{"A": vars["A"], "B": vars["B"], "C": vars["C"]},
					},
					result,
				})

				result = (vars["B"] || vars["A"]) && vars["C"]

				tests = append(tests, test{
					fmt.Sprintf("test 2 with %v+", vars),
					args{[]string{"((B", "OR", "A)", "AND", "C)"},
						map[string]bool{"A": vars["A"], "B": vars["B"], "C": vars["C"]},
					},
					result,
				})

				for _, b4 := range bools {
					vars["D"] = b4
					result := vars["A"] || ((vars["C"] && vars["B"]) || vars["D"]) && vars["B"]

					tests = append(tests, test{
						fmt.Sprintf("test 3 with %v+", vars),
						args{[]string{"A", "OR", "((C", "AND", "B)", "OR", "D)", "AND", "(B)"},
							map[string]bool{"A": vars["A"], "B": vars["B"], "C": vars["C"], "D": vars["D"]},
						},
						result,
					})

					result = vars["A"] || ((vars["C"] && vars["B"]) || (vars["D"]) && vars["B"])

					tests = append(tests, test{
						fmt.Sprintf("test 4 with %v+", vars),
						args{[]string{"A", "OR", "(((C", "AND", "B)", "OR", "(D", "AND", "(B))"},
							map[string]bool{"A": vars["A"], "B": vars["B"], "C": vars["C"], "D": vars["D"]},
						},
						result,
					})
				}
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := solveFormula(tt.args.formula, tt.args.vars)
			if got != tt.want {
				t.Errorf("solveFormula() got = %v, want %v", got, tt.want)
			}
		})
	}
}

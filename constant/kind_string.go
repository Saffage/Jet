// Code generated by "stringer -type=Kind"; DO NOT EDIT.

package constant

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Int-0]
	_ = x[Float-1]
	_ = x[String-2]
	_ = x[Bool-3]
}

const _Kind_name = "IntFloatStringBool"

var _Kind_index = [...]uint8{0, 3, 8, 14, 18}

func (i Kind) String() string {
	if i >= Kind(len(_Kind_index)-1) {
		return "Kind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Kind_name[_Kind_index[i]:_Kind_index[i+1]]
}
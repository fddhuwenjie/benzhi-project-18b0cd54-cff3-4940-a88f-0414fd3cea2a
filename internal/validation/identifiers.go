package validation

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func Identifier(name, value string) error {
	if err := Required(name, value); err != nil {
		return err
	}
	if !utf8.ValidString(value) {
		return invalid(name + " 必须是有效 UTF-8 文本")
	}
	if utf8.RuneCountInString(value) > 128 {
		return invalid(name + " 不得超过 128 个字符")
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_.:@/", r) {
			continue
		}
		return invalid(name + " 只能包含文字、数字及 -_.:@/ 分隔符")
	}
	return nil
}

func DistinctActors(leftName, left, rightName, right string) error {
	if err := Identifier(leftName, left); err != nil {
		return err
	}
	if err := Identifier(rightName, right); err != nil {
		return err
	}
	if left == right {
		return invalid(leftName + " 与 " + rightName + " 必须由不同人员承担")
	}
	return nil
}

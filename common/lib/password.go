package lib

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 对明文密码进行加密
// @author: [bluefrog](https://github.com/freewu)
// @function: HashPassword
// @description: 生成密码哈希
// @param: password string 明文密码
// @return: string 密码哈希, error
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 校验明文密码与密码哈希是否一致
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckPassword
// @description: 校验密码是否正确
// @param: hashedPassword string 密码哈希
// @param: password string 明文密码
// @return: bool
func CheckPassword(hashedPassword, password string) bool {
	if hashedPassword == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

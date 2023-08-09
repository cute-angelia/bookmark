package errorcode

type Code int //错误码

//go:generate stringer -type Code -linecomment
const (
	ErrorRegisterFailed          Code = 1000 // 用户注册失败
	ErrorLoginFailed             Code = 1001 // 登录失败
	ErrorBookmarkBase64Empty     Code = 2001 // 书签截图为空
	ErrorBookmarkBase64Error     Code = 2002 // 解码base64字符串获取图片数据错误
	ErrorBookmarkBase64WriteFile Code = 2003 // 将图片数据写入文件
	ErrorBookmarkBase64Decode    Code = 2004 // 编码图片信息
)

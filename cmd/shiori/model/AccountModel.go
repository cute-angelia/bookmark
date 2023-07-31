package model

// AccountModel is the database model for account.
type AccountModel struct {
	ID       int    `db:"id"       json:"id" gorm:"column:id"`
	Username string `db:"username" json:"username"     gorm:"column:username"`
	Password string `db:"password" json:"-" gorm:"column:password"`
	Owner    bool   `db:"owner"    json:"owner" gorm:"column:owner"`
}

func (AccountModel) TableName() string {
	return "account"
}

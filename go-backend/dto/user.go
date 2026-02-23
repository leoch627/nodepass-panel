package dto

type LoginDto struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaId     string `json:"captchaId"`
	CaptchaAnswer string `json:"captchaAnswer"`
}

type NodePermission struct {
	NodeId      int64 `json:"nodeId"`
	XrayEnabled *int  `json:"xrayEnabled"`
	GostEnabled *int  `json:"gostEnabled"`
}

type UserDto struct {
	User            string           `json:"user" binding:"required"`
	Pwd             string           `json:"pwd" binding:"required"`
	Flow            int64            `json:"flow"`
	XrayFlow        int64            `json:"xrayFlow"`
	Num             int              `json:"num"`
	ExpTime         int64            `json:"expTime"`
	FlowResetType   int              `json:"flowResetType"`
	FlowResetDay    int              `json:"flowResetDay"`
	Status          *int             `json:"status"`
	GostEnabled     *int             `json:"gostEnabled"`
	XrayEnabled     *int             `json:"xrayEnabled"`
	NodeIds         []int64          `json:"nodeIds"`
	NodePermissions []NodePermission `json:"nodePermissions"`
}

type UserUpdateDto struct {
	ID              int64            `json:"id" binding:"required"`
	User            string           `json:"user" binding:"required"`
	Pwd             string           `json:"pwd"`
	Flow            int64            `json:"flow"`
	XrayFlow        int64            `json:"xrayFlow"`
	Num             int              `json:"num"`
	ExpTime         int64            `json:"expTime"`
	FlowResetType   int              `json:"flowResetType"`
	FlowResetDay    int              `json:"flowResetDay"`
	Status          *int             `json:"status"`
	GostEnabled     *int             `json:"gostEnabled"`
	XrayEnabled     *int             `json:"xrayEnabled"`
	NodeIds         []int64          `json:"nodeIds"`
	NodePermissions []NodePermission `json:"nodePermissions"`
}

type UpdatePasswordDto struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
	NewUsername  string `json:"newUsername"`
}

type ResetFlowDto struct {
	ID int64 `json:"id" binding:"required"`
}

package condition

const (
	JWTSub Key = "jwt:sub"

	JWTIss Key = "jwt:iss"

	JWTAud Key = "jwt:aud"

	JWTJti Key = "jwt:jti"

	JWTName         Key = "jwt:name"
	JWTGivenName    Key = "jwt:given_name"
	JWTFamilyName   Key = "jwt:family_name"
	JWTMiddleName   Key = "jwt:middle_name"
	JWTNickName     Key = "jwt:nickname"
	JWTPrefUsername Key = "jwt:preferred_username"
	JWTProfile      Key = "jwt:profile"
	JWTPicture      Key = "jwt:picture"
	JWTWebsite      Key = "jwt:website"
	JWTEmail        Key = "jwt:email"
	JWTGender       Key = "jwt:gender"
	JWTBirthdate    Key = "jwt:birthdate"
	JWTPhoneNumber  Key = "jwt:phone_number"
	JWTAddress      Key = "jwt:address"
	JWTScope        Key = "jwt:scope"
	JWTClientID     Key = "jwt:client_id"
)

var JWTKeys = []Key{
	JWTSub,
	JWTIss,
	JWTAud,
	JWTJti,
	JWTName,
	JWTGivenName,
	JWTFamilyName,
	JWTMiddleName,
	JWTNickName,
	JWTPrefUsername,
	JWTProfile,
	JWTPicture,
	JWTWebsite,
	JWTEmail,
	JWTGender,
	JWTBirthdate,
	JWTPhoneNumber,
	JWTAddress,
	JWTScope,
	JWTClientID,
}

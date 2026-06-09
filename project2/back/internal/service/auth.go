package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/auth"
	"github.com/tochka-pamyati/tochka-pamyati/internal/config"
	"github.com/tochka-pamyati/tochka-pamyati/internal/ids"
	"github.com/tochka-pamyati/tochka-pamyati/internal/mailer"
	"github.com/tochka-pamyati/tochka-pamyati/internal/repo"
)

const AccessCookieName = "tp_access"
const RefreshCookieName = "tp_refresh"
const verificationPurposeEmail = "verify_email"
const verificationPurposePasswordReset = "password_reset"

type Auth struct {
	cfg       config.AuthConfig
	users     *repo.Users
	roles     *repo.Roles
	verifs    *repo.EmailVerifications
	sess      *repo.Sessions
	adminLogs *repo.AdminEventLogs
	sanctions *SanctionsService
	trust     *TrustService

	mailer    mailer.Sender
	mailQueue *mailer.Queue
}

type AuthDeps struct {
	Config             config.AuthConfig
	Users              *repo.Users
	Roles              *repo.Roles
	EmailVerifications *repo.EmailVerifications
	Sessions           *repo.Sessions
	AdminLogs          *repo.AdminEventLogs
	Sanctions          *SanctionsService
	Trust              *TrustService
	Mailer             mailer.Sender
	MailQueue          *mailer.Queue
}

func NewAuth(deps AuthDeps) *Auth {
	return &Auth{
		cfg:       deps.Config,
		users:     deps.Users,
		roles:     deps.Roles,
		verifs:    deps.EmailVerifications,
		sess:      deps.Sessions,
		adminLogs: deps.AdminLogs,
		sanctions: deps.Sanctions,
		trust:     deps.Trust,
		mailer:    deps.Mailer,
		mailQueue: deps.MailQueue,
	}
}

func (a *Auth) logEvent(ctx context.Context, actorID, targetUserID *uuid.UUID, entityType string, entityID *uuid.UUID, action, message string, meta map[string]any) {
	if a.adminLogs == nil {
		return
	}
	_, _ = a.adminLogs.Create(ctx, repo.CreateAdminEventLogParams{
		ActorUserID:  actorID,
		TargetUserID: targetUserID,
		EntityType:   entityType,
		EntityID:     entityID,
		Action:       action,
		Result:       "success",
		Message:      message,
		Meta:         meta,
	})
}

type PublicUser struct {
	ID                   uuid.UUID           `json:"id"`
	Username             string              `json:"username"`
	Email                string              `json:"email"`
	RoleID               uuid.UUID           `json:"role_id"`
	RoleName             string              `json:"role_name"`
	TrustScore           int                 `json:"trust_score"`
	City                 string              `json:"city,omitempty"`
	Region               string              `json:"region,omitempty"`
	NotificationSettings map[string]any      `json:"notification_settings"`
	IsActive             bool                `json:"is_active"`
	IsBlocked            bool                `json:"is_blocked"`
	ActiveSanctions      []repo.UserSanction `json:"active_sanctions,omitempty"`
	RestrictionSummary   *RestrictionSummary `json:"restriction_summary,omitempty"`
	TrustSummary         *TrustSummary       `json:"trust_summary,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
	LastLogin            *time.Time          `json:"last_login,omitempty"`
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
	Consent  bool
}

type RegisterOutput struct {
	User PublicUser `json:"user"`
}

func (a *Auth) Register(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	fields := map[string]string{}
	in.Username = strings.TrimSpace(in.Username)
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Username == "" {
		fields["username"] = "required"
	}
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		fields["email"] = "invalid"
	}
	mergeFieldErrors(fields, validatePasswordFields(in.Password))
	if !in.Consent {
		fields["consent"] = "required"
	}
	if len(fields) > 0 {
		return RegisterOutput{}, apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: fields}
	}

	roleID, err := a.roles.GetByName(ctx, "user")
	if err != nil {
		return RegisterOutput{}, err
	}

	hash, err := auth.HashArgon2idPHC(in.Password, auth.DefaultArgon2idParams())
	if err != nil {
		return RegisterOutput{}, err
	}

	user, err := a.users.Create(ctx, repo.User{
		Username:             in.Username,
		Email:                in.Email,
		PasswordHash:         hash,
		RoleID:               roleID,
		TrustScore:           5,
		NotificationSettings: map[string]any{},
		IsActive:             false,
		IsBlocked:            false,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "users_email_key") || strings.Contains(strings.ToLower(err.Error()), "email") && strings.Contains(strings.ToLower(err.Error()), "unique") {
			return RegisterOutput{}, apierr.Error{Code: "email_already_used", Message: "email already used"}
		}
		if strings.Contains(strings.ToLower(err.Error()), "users_username_key") || strings.Contains(strings.ToLower(err.Error()), "username") && strings.Contains(strings.ToLower(err.Error()), "unique") {
			return RegisterOutput{}, apierr.Error{Code: "username_already_used", Message: "username already used"}
		}
		return RegisterOutput{}, err
	}

	code, err := gen6()
	if err != nil {
		return RegisterOutput{}, err
	}
	expiresAt := time.Now().Add(2 * time.Hour)
	_, err = a.verifs.CreateWithPurpose(ctx, user.ID, user.Email, verificationPurposeEmail, code, expiresAt)
	if err != nil {
		return RegisterOutput{}, err
	}

	a.mailQueue.Enqueue(a.mailer, mailer.Message{
		To:      user.Email,
		Subject: "Код подтверждения — Точка памяти",
		Body:    buildVerificationHTML(code),
		IsHTML:  true,
	})

	a.logEvent(ctx, &user.ID, &user.ID, "user", &user.ID, "регистрация", "Пользователь зарегистрировал новый аккаунт", map[string]any{
		"email": user.Email,
	})

	return RegisterOutput{User: a.toPublicUserWithRole(ctx, user)}, nil
}

type VerifyEmailInput struct {
	Email string
	Code  string
}

func (a *Auth) VerifyEmail(ctx context.Context, in VerifyEmailInput) error {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.Code = strings.TrimSpace(in.Code)
	if in.Email == "" || len(in.Code) != 6 {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"email": "invalid", "code": "invalid"}}
	}

	v, err := a.verifs.LatestActiveByEmailAndPurpose(ctx, in.Email, verificationPurposeEmail)
	if err != nil {
		return apierr.Error{Code: "verification_invalid", Message: "invalid code"}
	}

	if time.Now().After(v.ExpiresAt) {
		return apierr.Error{Code: "verification_expired", Message: "code expired"}
	}

	attempts, err := a.verifs.IncrementAttempts(ctx, v.ID)
	if err != nil {
		return err
	}
	if attempts > 10 {
		return apierr.Error{Code: "rate_limited", Message: "too many attempts"}
	}

	if subtleConstantTimeStringEq(v.Code, in.Code) == false {
		return apierr.Error{Code: "verification_invalid", Message: "invalid code"}
	}

	now := time.Now()
	if err := a.verifs.MarkUsed(ctx, v.ID, now); err != nil {
		return err
	}
	if err := a.users.Activate(ctx, v.UserID); err != nil {
		return err
	}
	return nil
}

type ResendCodeInput struct {
	Email string
}

func (a *Auth) ResendCode(ctx context.Context, in ResendCodeInput) error {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"email": "required"}}
	}

	now := time.Now()
	sentLast2Min, sentToday, err := a.verifs.ResendLimitsByPurpose(ctx, in.Email, verificationPurposeEmail, now)
	if err != nil {
		return err
	}
	if sentLast2Min || sentToday >= 5 {
		return apierr.Error{Code: "rate_limited", Message: "resend limit reached"}
	}

	user, err := a.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil
	}
	if user.IsActive {
		return nil
	}

	code, err := gen6()
	if err != nil {
		return err
	}
	expiresAt := now.Add(2 * time.Hour)
	_, err = a.verifs.CreateWithPurpose(ctx, user.ID, user.Email, verificationPurposeEmail, code, expiresAt)
	if err != nil {
		return err
	}

	a.mailQueue.Enqueue(a.mailer, mailer.Message{
		To:      user.Email,
		Subject: "Код подтверждения — Точка памяти",
		Body:    buildVerificationHTML(code),
		IsHTML:  true,
	})

	return nil
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	User          PublicUser
	AccessCookie  *http.Cookie
	RefreshCookie *http.Cookie
}

func (a *Auth) Login(ctx context.Context, in LoginInput, ip, userAgent string) (LoginOutput, error) {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || in.Password == "" {
		return LoginOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid credentials"}
	}

	user, err := a.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return LoginOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid credentials"}
	}
	if user.IsBlocked {
		return LoginOutput{}, apierr.Error{Code: "account_blocked", Message: "Вход в аккаунт заблокирован"}
	}
	if a.sanctions != nil {
		if err := a.sanctions.Check(ctx, user.ID, SanctionScopeLogin); err != nil {
			return LoginOutput{}, err
		}
	}
	if !user.IsActive {
		return LoginOutput{}, apierr.Error{Code: "email_not_verified", Message: "email not verified"}
	}

	if !auth.VerifyArgon2idPHC(user.PasswordHash, in.Password) {
		return LoginOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid credentials"}
	}

	now := time.Now()
	_ = a.users.SetLastLogin(ctx, user.ID, now)

	accessToken, err := auth.SignHS256(a.cfg.JWTSecret, user.ID, "access", now.Add(a.cfg.AccessTokenTTL), uuid.Nil)
	if err != nil {
		return LoginOutput{}, err
	}

	refreshJTI := ids.NewV7()
	refreshToken, err := auth.SignHS256(a.cfg.JWTSecret, user.ID, "refresh", now.Add(a.cfg.RefreshTokenTTL), refreshJTI)
	if err != nil {
		return LoginOutput{}, err
	}

	if err := a.sess.Create(ctx, user.ID, refreshJTI, now.Add(a.cfg.RefreshTokenTTL), ip, userAgent); err != nil {
		return LoginOutput{}, err
	}

	a.logEvent(ctx, &user.ID, &user.ID, "user", &user.ID, "вход_в_систему", "Пользователь вошел в систему", map[string]any{
		"ip":         ip,
		"user_agent": userAgent,
	})

	return LoginOutput{
		User:          a.toPublicUserWithRole(ctx, user),
		AccessCookie:  buildAuthCookie(AccessCookieName, accessToken, now.Add(a.cfg.AccessTokenTTL)),
		RefreshCookie: buildAuthCookie(RefreshCookieName, refreshToken, now.Add(a.cfg.RefreshTokenTTL)),
	}, nil
}

type RefreshOutput struct {
	User          PublicUser
	AccessCookie  *http.Cookie
	RefreshCookie *http.Cookie
}

func (a *Auth) Refresh(ctx context.Context, refreshToken string) (RefreshOutput, error) {
	claims, err := auth.ParseHS256(a.cfg.JWTSecret, refreshToken, "refresh")
	if err != nil {
		return RefreshOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid refresh token"}
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return RefreshOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid refresh token"}
	}
	refreshJTI, err := uuid.Parse(claims.ID)
	if err != nil {
		return RefreshOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid refresh token"}
	}

	if _, err := a.sess.GetActiveByJTI(ctx, refreshJTI); err != nil {
		return RefreshOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid refresh token"}
	}

	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return RefreshOutput{}, apierr.Error{Code: "invalid_credentials", Message: "invalid refresh token"}
	}
	if user.IsBlocked {
		return RefreshOutput{}, apierr.Error{Code: "account_blocked", Message: "Вход в аккаунт заблокирован"}
	}
	if a.sanctions != nil {
		if err := a.sanctions.Check(ctx, user.ID, SanctionScopeLogin); err != nil {
			_ = a.sess.RevokeByUserID(ctx, user.ID, time.Now())
			return RefreshOutput{}, err
		}
	}

	now := time.Now()
	accessToken, err := auth.SignHS256(a.cfg.JWTSecret, user.ID, "access", now.Add(a.cfg.AccessTokenTTL), uuid.Nil)
	if err != nil {
		return RefreshOutput{}, err
	}

	newRefreshJTI := ids.NewV7()
	newRefreshToken, err := auth.SignHS256(a.cfg.JWTSecret, user.ID, "refresh", now.Add(a.cfg.RefreshTokenTTL), newRefreshJTI)
	if err != nil {
		return RefreshOutput{}, err
	}

	_ = a.sess.RevokeByJTI(ctx, refreshJTI, now)
	if err := a.sess.Create(ctx, user.ID, newRefreshJTI, now.Add(a.cfg.RefreshTokenTTL), "", ""); err != nil {
		return RefreshOutput{}, err
	}

	return RefreshOutput{
		User:          a.toPublicUserWithRole(ctx, user),
		AccessCookie:  buildAuthCookie(AccessCookieName, accessToken, now.Add(a.cfg.AccessTokenTTL)),
		RefreshCookie: buildAuthCookie(RefreshCookieName, newRefreshToken, now.Add(a.cfg.RefreshTokenTTL)),
	}, nil
}

func (a *Auth) Logout(ctx context.Context, refreshToken string) error {
	claims, err := auth.ParseHS256(a.cfg.JWTSecret, refreshToken, "refresh")
	if err != nil {
		return nil
	}
	refreshJTI, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil
	}
	return a.sess.RevokeByJTI(ctx, refreshJTI, time.Now())
}

func (a *Auth) Me(ctx context.Context, accessToken string) (PublicUser, error) {
	claims, err := auth.ParseHS256(a.cfg.JWTSecret, accessToken, "access")
	if err != nil {
		return PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}
	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}

	// Получаем имя роли для фронтенда
	roleName, _ := a.roles.GetNameByID(ctx, user.RoleID)
	p := toPublicUser(user)
	p.RoleName = roleName
	return a.enrichPublicUser(ctx, p, user), nil
}

type UpdateProfileInput struct {
	Username             string
	City                 string
	Region               string
	NotificationSettings map[string]any
}

func (a *Auth) UpdateProfile(ctx context.Context, accessToken string, in UpdateProfileInput) (PublicUser, error) {
	claims, err := auth.ParseHS256(a.cfg.JWTSecret, accessToken, "access")
	if err != nil {
		return PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return PublicUser{}, apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}

	fields := map[string]string{}
	in.Username = strings.TrimSpace(in.Username)
	in.City = strings.TrimSpace(in.City)
	in.Region = strings.TrimSpace(in.Region)
	if in.Username == "" {
		fields["username"] = "required"
	}
	if len(fields) > 0 {
		return PublicUser{}, apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: fields}
	}

	settings := in.NotificationSettings
	if settings == nil {
		settings = map[string]any{}
	}

	user, err := a.users.UpdateProfile(ctx, userID, in.Username, in.City, in.Region, settings)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "users_username_key") || strings.Contains(strings.ToLower(err.Error()), "username") && strings.Contains(strings.ToLower(err.Error()), "unique") {
			return PublicUser{}, apierr.Error{Code: "username_already_used", Message: "username already used"}
		}
		return PublicUser{}, err
	}
	a.logEvent(ctx, &userID, &userID, "user", &userID, "обновление_профиля", "Пользователь обновил настройки профиля", map[string]any{
		"city":   in.City,
		"region": in.Region,
	})
	return a.toPublicUserWithRole(ctx, user), nil
}

type ChangePasswordInput struct {
	OldPassword string
	NewPassword string
}

func (a *Auth) ChangePassword(ctx context.Context, accessToken string, in ChangePasswordInput) error {
	claims, err := auth.ParseHS256(a.cfg.JWTSecret, accessToken, "access")
	if err != nil {
		return apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}
	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		return apierr.Error{Code: "invalid_credentials", Message: "invalid access token"}
	}

	fields := map[string]string{}
	if strings.TrimSpace(in.OldPassword) == "" {
		fields["old_password"] = "required"
	}
	mergeFieldErrors(fields, validatePasswordFields(in.NewPassword))
	if len(fields) > 0 {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: fields}
	}
	if !auth.VerifyArgon2idPHC(user.PasswordHash, in.OldPassword) {
		return apierr.Error{Code: "invalid_credentials", Message: "invalid current password"}
	}
	if in.OldPassword == in.NewPassword {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"new_password": "same_as_old"}}
	}

	hash, err := auth.HashArgon2idPHC(in.NewPassword, auth.DefaultArgon2idParams())
	if err != nil {
		return err
	}
	if err := a.users.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	_ = a.sess.RevokeByUserID(ctx, userID, time.Now())
	a.logEvent(ctx, &userID, &userID, "user", &userID, "смена_пароля", "Пользователь сменил пароль", nil)
	return nil
}

type RequestPasswordResetInput struct {
	Email string
}

func (a *Auth) RequestPasswordReset(ctx context.Context, in RequestPasswordResetInput) error {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: map[string]string{"email": "required"}}
	}

	now := time.Now()
	sentLast2Min, sentToday, err := a.verifs.ResendLimitsByPurpose(ctx, in.Email, verificationPurposePasswordReset, now)
	if err != nil {
		return err
	}
	if sentLast2Min || sentToday >= 5 {
		return apierr.Error{Code: "rate_limited", Message: "resend limit reached"}
	}

	user, err := a.users.GetByEmail(ctx, in.Email)
	if err != nil || !user.IsActive {
		return nil
	}

	code, err := gen6()
	if err != nil {
		return err
	}
	expiresAt := now.Add(30 * time.Minute)
	if _, err := a.verifs.CreateWithPurpose(ctx, user.ID, user.Email, verificationPurposePasswordReset, code, expiresAt); err != nil {
		return err
	}

	a.mailQueue.Enqueue(a.mailer, mailer.Message{
		To:      user.Email,
		Subject: "Код восстановления пароля — Точка памяти",
		Body:    buildPasswordResetHTML(code),
		IsHTML:  true,
	})
	a.logEvent(ctx, &user.ID, &user.ID, "user", &user.ID, "запрос_сброса_пароля", "Пользователь запросил код для сброса пароля", nil)
	return nil
}

type ResetPasswordInput struct {
	Email       string
	Code        string
	NewPassword string
}

func (a *Auth) ResetPassword(ctx context.Context, in ResetPasswordInput) error {
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	in.Code = strings.TrimSpace(in.Code)

	fields := map[string]string{}
	if in.Email == "" || !strings.Contains(in.Email, "@") {
		fields["email"] = "invalid"
	}
	if len(in.Code) != 6 {
		fields["code"] = "invalid"
	}
	mergeFieldErrors(fields, validatePasswordFields(in.NewPassword))
	if len(fields) > 0 {
		return apierr.Error{Code: "validation_failed", Message: "invalid input", Fields: fields}
	}

	v, err := a.verifs.LatestActiveByEmailAndPurpose(ctx, in.Email, verificationPurposePasswordReset)
	if err != nil {
		return apierr.Error{Code: "verification_invalid", Message: "invalid code"}
	}
	if time.Now().After(v.ExpiresAt) {
		return apierr.Error{Code: "verification_expired", Message: "code expired"}
	}

	attempts, err := a.verifs.IncrementAttempts(ctx, v.ID)
	if err != nil {
		return err
	}
	if attempts > 10 {
		return apierr.Error{Code: "rate_limited", Message: "too many attempts"}
	}
	if !subtleConstantTimeStringEq(v.Code, in.Code) {
		return apierr.Error{Code: "verification_invalid", Message: "invalid code"}
	}

	user, err := a.users.GetByID(ctx, v.UserID)
	if err != nil {
		return apierr.Error{Code: "verification_invalid", Message: "invalid code"}
	}
	hash, err := auth.HashArgon2idPHC(in.NewPassword, auth.DefaultArgon2idParams())
	if err != nil {
		return err
	}
	if err := a.users.UpdatePasswordHash(ctx, user.ID, hash); err != nil {
		return err
	}
	now := time.Now()
	if err := a.verifs.MarkUsed(ctx, v.ID, now); err != nil {
		return err
	}
	_ = a.sess.RevokeByUserID(ctx, user.ID, now)
	a.logEvent(ctx, &user.ID, &user.ID, "user", &user.ID, "сброс_пароля", "Пользователь завершил смену пароля по коду из письма", nil)
	return nil
}

func buildAuthCookie(name, value string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  expires,
	}
}

func ExpiredCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func gen6() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if n < 0 {
		n = -n
	}
	return fmt.Sprintf("%06d", n%1000000), nil
}

func subtleConstantTimeStringEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func (a *Auth) toPublicUserWithRole(ctx context.Context, u repo.User) PublicUser {
	p := toPublicUser(u)
	if name, err := a.roles.GetNameByID(ctx, u.RoleID); err == nil {
		p.RoleName = name
	}
	return a.enrichPublicUser(ctx, p, u)
}

func toPublicUser(u repo.User) PublicUser {
	return PublicUser{
		ID:                   u.ID,
		Username:             u.Username,
		Email:                u.Email,
		RoleID:               u.RoleID,
		TrustScore:           u.TrustScore,
		City:                 u.City,
		Region:               u.Region,
		NotificationSettings: u.NotificationSettings,
		IsActive:             u.IsActive,
		IsBlocked:            u.IsBlocked,
		CreatedAt:            u.CreatedAt,
		LastLogin:            u.LastLogin,
	}
}

func (a *Auth) enrichPublicUser(ctx context.Context, p PublicUser, u repo.User) PublicUser {
	if a.sanctions == nil {
		if a.trust != nil {
			if trustSummary, err := a.trust.Summary(ctx, u); err == nil {
				p.TrustSummary = &trustSummary
			}
		}
		return p
	}
	summary, err := a.sanctions.Summary(ctx, u)
	if err != nil {
		return p
	}
	p.ActiveSanctions = summary.ActiveSanctions
	if summary.Status != "active" || len(summary.ActiveSanctions) > 0 || u.IsBlocked {
		p.RestrictionSummary = &summary
	}
	if a.trust != nil {
		if trustSummary, err := a.trust.Summary(ctx, u); err == nil {
			p.TrustSummary = &trustSummary
		}
	}
	return p
}

func buildVerificationHTML(code string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <style>
        body { margin: 0; padding: 0; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
        .wrapper { padding: 40px 20px; }
        .container { max-width: 500px; margin: 0 auto; background: #ffffff; border-radius: 40px; padding: 40px; box-shadow: 0 20px 50px rgba(0,0,0,0.05); border: 1px solid #f1f5f9; }
        .logo { background: rgba(225, 29, 72, 0.1); width: 64px; height: 64px; border-radius: 20px; margin: 0 auto 24px; display: flex; align-items: center; justify-content: center; }
        .logo-icon { color: #e11d48; font-size: 32px; font-weight: bold; line-height: 64px; text-align: center; width: 100%%; }
        h1 { font-size: 32px; font-weight: 900; text-align: center; color: #0f172a; margin: 0 0 8px; letter-spacing: -1px; }
        p { font-size: 16px; font-weight: 500; text-align: center; color: #64748b; margin: 0 0 32px; line-height: 1.5; }
        .code-container { background: #f1f5f9; border-radius: 24px; padding: 32px; text-align: center; margin-bottom: 32px; }
        .code { font-size: 48px; font-weight: 900; color: #e11d48; letter-spacing: 8px; margin-left: 8px; }
        .footer { text-align: center; font-size: 14px; font-weight: 600; color: #94a3b8; text-transform: uppercase; letter-spacing: 1px; }
        .divider { height: 1px; background: #f1f5f9; margin: 32px 0; }
    </style>
</head>
<body>
    <div class="wrapper">
        <div class="container">
            <div class="logo">
                <div class="logo-icon">ТП</div>
            </div>
            <h1>Подтверждение</h1>
            <p>Мы отправили 6-значный код для активации вашего аккаунта в проекте <b>Точка памяти</b></p>
            
            <div class="code-container">
                <div class="code">%s</div>
            </div>
            
            <p style="font-size: 14px; margin-bottom: 0;">Код действует 2 часа. Если вы не запрашивали регистрацию, просто удалите это письмо.</p>
            
            <div class="divider"></div>
            
            <div class="footer">
                Точка памяти &copy; 2026
            </div>
        </div>
    </div>
</body>
</html>
`, code)
}

func hasUppercase(password string) bool {
	for _, r := range password {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func hasSpecialCharacter(password string) bool {
	for _, r := range password {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func validatePasswordFields(password string) map[string]string {
	fields := map[string]string{}
	if len(password) < 8 {
		fields["password"] = "min_8"
	}
	if !hasUppercase(password) {
		fields["password_uppercase"] = "missing_uppercase"
	}
	if !hasSpecialCharacter(password) {
		fields["password_special"] = "missing_special"
	}
	return fields
}

func mergeFieldErrors(target map[string]string, extra map[string]string) {
	for key, value := range extra {
		target[key] = value
	}
}

func buildPasswordResetHTML(code string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <style>
        body { margin: 0; padding: 0; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
        .wrapper { padding: 40px 20px; }
        .container { max-width: 500px; margin: 0 auto; background: #ffffff; border-radius: 40px; padding: 40px; box-shadow: 0 20px 50px rgba(0,0,0,0.05); border: 1px solid #f1f5f9; }
        .logo { background: rgba(37, 99, 235, 0.1); width: 64px; height: 64px; border-radius: 20px; margin: 0 auto 24px; display: flex; align-items: center; justify-content: center; }
        .logo-icon { color: #2563eb; font-size: 32px; font-weight: bold; line-height: 64px; text-align: center; width: 100%%; }
        h1 { font-size: 32px; font-weight: 900; text-align: center; color: #0f172a; margin: 0 0 8px; letter-spacing: -1px; }
        p { font-size: 16px; font-weight: 500; text-align: center; color: #64748b; margin: 0 0 32px; line-height: 1.5; }
        .code-container { background: #eff6ff; border-radius: 24px; padding: 32px; text-align: center; margin-bottom: 32px; }
        .code { font-size: 48px; font-weight: 900; color: #2563eb; letter-spacing: 8px; margin-left: 8px; }
        .footer { text-align: center; font-size: 14px; font-weight: 600; color: #94a3b8; text-transform: uppercase; letter-spacing: 1px; }
        .divider { height: 1px; background: #f1f5f9; margin: 32px 0; }
    </style>
</head>
<body>
    <div class="wrapper">
        <div class="container">
            <div class="logo">
                <div class="logo-icon">ТП</div>
            </div>
            <h1>Сброс пароля</h1>
            <p>Поступил запрос на восстановление доступа к аккаунту в проекте <b>Точка памяти</b>. Для смены пароля используйте этот код.</p>
            <div class="code-container">
                <div class="code">%s</div>
            </div>
            <p style="font-size: 14px; margin-bottom: 0;">Код действует 30 минут. Если запрос отправлялся не владельцем аккаунта, письмо можно просто удалить.</p>
            <div class="divider"></div>
            <div class="footer">
                Точка памяти &copy; 2026
            </div>
        </div>
    </div>
</body>
</html>
`, code)
}

package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/tochka-pamyati/tochka-pamyati/internal/apierr"
	"github.com/tochka-pamyati/tochka-pamyati/internal/service"
)

type AuthHandler struct {
	auth *service.Auth
}

func NewAuthHandler(auth *service.Auth) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Consent  bool   `json:"consent"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	out, err := h.auth.Register(c.Request().Context(), service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Consent:  req.Consent,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, out)
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	var req VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.auth.VerifyEmail(c.Request().Context(), service.VerifyEmailInput{Email: req.Email, Code: req.Code}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type ResendCodeRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) ResendCode(c echo.Context) error {
	var req ResendCodeRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.auth.ResendCode(c.Request().Context(), service.ResendCodeInput{Email: req.Email}); err != nil {
		return err
	}
	return c.NoContent(http.StatusAccepted)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	out, err := h.auth.Login(c.Request().Context(), service.LoginInput{Email: req.Email, Password: req.Password}, c.RealIP(), c.Request().UserAgent())
	if err != nil {
		return err
	}

	http.SetCookie(c.Response(), out.AccessCookie)
	http.SetCookie(c.Response(), out.RefreshCookie)
	return c.JSON(http.StatusOK, out.User)
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	refreshCookie, err := c.Cookie(service.RefreshCookieName)
	if err != nil || refreshCookie.Value == "" {
		return apierr.Error{Code: "invalid_credentials", Message: "missing refresh token"}
	}

	out, err := h.auth.Refresh(c.Request().Context(), refreshCookie.Value)
	if err != nil {
		return err
	}

	http.SetCookie(c.Response(), out.AccessCookie)
	http.SetCookie(c.Response(), out.RefreshCookie)
	return c.JSON(http.StatusOK, out.User)
}

func (h *AuthHandler) Logout(c echo.Context) error {
	refreshCookie, _ := c.Cookie(service.RefreshCookieName)
	if refreshCookie != nil && refreshCookie.Value != "" {
		_ = h.auth.Logout(c.Request().Context(), refreshCookie.Value)
	}

	http.SetCookie(c.Response(), service.ExpiredCookie(service.AccessCookieName))
	http.SetCookie(c.Response(), service.ExpiredCookie(service.RefreshCookieName))
	return c.NoContent(http.StatusNoContent)
}

func (h *AuthHandler) Me(c echo.Context) error {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || accessCookie.Value == "" {
		return apierr.Error{Code: "invalid_credentials", Message: "missing access token"}
	}
	user, err := h.auth.Me(c.Request().Context(), accessCookie.Value)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

type UpdateProfileRequest struct {
	Username             string         `json:"username"`
	City                 string         `json:"city"`
	Region               string         `json:"region"`
	NotificationSettings map[string]any `json:"notification_settings"`
}

func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || accessCookie.Value == "" {
		return apierr.Error{Code: "invalid_credentials", Message: "missing access token"}
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}

	user, err := h.auth.UpdateProfile(c.Request().Context(), accessCookie.Value, service.UpdateProfileInput{
		Username:             req.Username,
		City:                 req.City,
		Region:               req.Region,
		NotificationSettings: req.NotificationSettings,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
	accessCookie, err := c.Cookie(service.AccessCookieName)
	if err != nil || accessCookie.Value == "" {
		return apierr.Error{Code: "invalid_credentials", Message: "missing access token"}
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.auth.ChangePassword(c.Request().Context(), accessCookie.Value, service.ChangePasswordInput{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}); err != nil {
		return err
	}
	http.SetCookie(c.Response(), service.ExpiredCookie(service.RefreshCookieName))
	return c.NoContent(http.StatusNoContent)
}

type RequestPasswordResetRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) RequestPasswordReset(c echo.Context) error {
	var req RequestPasswordResetRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.auth.RequestPasswordReset(c.Request().Context(), service.RequestPasswordResetInput{Email: req.Email}); err != nil {
		return err
	}
	return c.NoContent(http.StatusAccepted)
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return apierr.Error{Code: "validation_failed", Message: "invalid payload"}
	}
	if err := h.auth.ResetPassword(c.Request().Context(), service.ResetPasswordInput{
		Email:       req.Email,
		Code:        req.Code,
		NewPassword: req.NewPassword,
	}); err != nil {
		return err
	}
	http.SetCookie(c.Response(), service.ExpiredCookie(service.AccessCookieName))
	http.SetCookie(c.Response(), service.ExpiredCookie(service.RefreshCookieName))
	return c.NoContent(http.StatusNoContent)
}

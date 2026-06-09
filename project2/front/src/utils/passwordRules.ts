export interface PasswordRuleState {
  minLength: boolean;
  uppercase: boolean;
  special: boolean;
}

export const getPasswordRuleState = (password: string): PasswordRuleState => ({
  minLength: password.length >= 8,
  uppercase: /[A-ZА-ЯЁ]/.test(password),
  special: /[^A-Za-zА-Яа-яЁё0-9\s]/.test(password),
});

export const isPasswordStrongEnough = (password: string) => {
  const state = getPasswordRuleState(password);
  return state.minLength && state.uppercase && state.special;
};

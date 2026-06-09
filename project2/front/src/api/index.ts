import axios from 'axios';

const fieldNameLabel = (field: string) => {
  const labels: Record<string, string> = {
    username: 'Никнейм',
    email: 'Электронная почта',
    password: 'Пароль',
    old_password: 'Текущий пароль',
    new_password: 'Новый пароль',
    consent: 'Согласие',
    content: 'Текст',
    description: 'Описание',
    comment: 'Комментарий',
    reason: 'Причина',
    urgency: 'Срочность',
    q: 'Поисковый запрос',
    entity_id: 'Идентификатор объекта',
    monument_id: 'Памятник',
    signal_id: 'Сигнал',
    comment_id: 'Комментарий',
    post_id: 'Пост',
    target_part_id: 'Фотография',
    duplicate_target_id: 'Памятник для объединения',
    edited_content: 'Исправленный текст',
  };
  return labels[field] || 'Поле';
};

const fieldRuleLabel = (field: string, rule: string) => {
  if (field === 'password' && rule === 'min_8') return 'Пароль должен содержать не менее 8 символов.';
  if ((field === 'password' || field === 'new_password' || field === 'password_uppercase' || field === 'new_password_uppercase') && rule === 'missing_uppercase') {
    return 'Пароль должен содержать хотя бы одну заглавную букву.';
  }
  if ((field === 'password' || field === 'new_password' || field === 'password_special' || field === 'new_password_special') && rule === 'missing_special') {
    return 'Пароль должен содержать хотя бы один специальный символ.';
  }
  if (field === 'new_password' && rule === 'same_as_old') return 'Новый пароль должен отличаться от текущего.';
  if (field === 'email' && rule === 'invalid') return 'Укажите корректный адрес электронной почты.';
  if (field === 'username' && rule === 'required') return 'Укажите никнейм.';
  if (field === 'email' && rule === 'required') return 'Укажите адрес электронной почты.';
  if (field === 'consent' && rule === 'required') return 'Нужно согласиться на обработку персональных данных.';
  if (field === 'description' && rule === 'required_or_photos') return 'Нужно добавить описание или фотографии.';
  if (field === 'photos' && rule === 'max_10') return 'Можно прикрепить не более 10 фотографий.';
  if (rule === 'required') return `${fieldNameLabel(field)}: поле обязательно для заполнения.`;
  if (rule === 'invalid') return `${fieldNameLabel(field)}: указано некорректное значение.`;
  if (rule === 'invalid_json') return `${fieldNameLabel(field)}: некорректный формат данных.`;
  if (rule === 'not_found') return `${fieldNameLabel(field)}: объект не найден.`;
  if (rule === 'required_or_photos') return `${fieldNameLabel(field)}: нужно заполнить поле или добавить фотографии.`;
  if (rule === 'max_10') return `${fieldNameLabel(field)}: можно добавить не более 10 фотографий.`;
  return `${fieldNameLabel(field)}: требуется проверка значения.`;
};

const translateFields = (fields?: Record<string, string>) => {
  if (!fields || typeof fields !== 'object') return '';
  const messages = Object.entries(fields).map(([field, rule]) => fieldRuleLabel(field, String(rule)));
  return messages.join(' ');
};

const translateApiMessage = (message?: string, fields?: Record<string, string>) => {
  const rawMessage = String(message || '').trim();
  const normalized = rawMessage.toLowerCase();
  const fieldDetails = translateFields(fields);

  if (normalized === 'invalid input' || normalized === 'invalid payload') {
    return fieldDetails || 'Некорректно заполнены поля формы.';
  }
  if (normalized === 'email already used') return 'Такой адрес электронной почты уже зарегистрирован.';
  if (normalized === 'username already used') return 'Такой никнейм уже занят.';
  if (normalized === 'cannot report own content') return 'Нельзя жаловаться на собственный контент.';
  if (normalized === 'trust_content_create_blocked') return 'Уровень доверия сейчас слишком низкий: создание нового контента временно недоступно.';
  if (normalized === 'trust_comment_create_blocked') return 'Уровень доверия сейчас слишком низкий: комментарии временно недоступны.';
  if (normalized === 'trust_edit_blocked') return 'Уровень доверия сейчас слишком низкий: редактирование временно недоступно.';
  if (normalized === 'trust_rate_limited') return 'Для текущего уровня доверия временно достигнут лимит активности. Часть действий станет доступна позже.';
  if (normalized === 'trust_premoderation_required') return 'Новый материал будет отправлен на обязательную проверку из-за текущего уровня доверия.';
  if (normalized === 'missing query') return 'Нужно ввести поисковый запрос.';
  if (normalized === 'invalid user id') return 'Некорректный идентификатор пользователя.';
  if (normalized === 'invalid sanction id') return 'Некорректный идентификатор ограничения.';
  if (normalized === 'invalid notification id') return 'Некорректный идентификатор уведомления.';
  if (normalized === 'invalid monument id') return 'Некорректный идентификатор памятника.';
  if (normalized === 'invalid signal id') return 'Некорректный идентификатор сигнала.';
  if (normalized === 'invalid comment id') return 'Некорректный идентификатор комментария.';
  if (normalized === 'invalid post id') return 'Некорректный идентификатор поста.';
  if (normalized === 'invalid entity type') return 'Некорректный тип объекта.';
  if (normalized === 'unsupported entity type') return 'Неподдерживаемый тип объекта.';
  if (normalized === 'invalid action') return 'Недопустимое действие.';
  if (normalized === 'confirmation required') return 'Нужно подтвердить действие.';
  if (normalized === 'invalid current password') return 'Текущий пароль указан неверно.';
  if (normalized === 'reason required') return 'Нужно указать причину.';
  if (normalized === 'comment required for rejection') return 'Нужно указать комментарий для отклонения.';
  if (normalized === 'signal not found') return 'Сигнал не найден.';
  if (normalized === 'monument not found') return 'Памятник не найден.';
  if (normalized === 'post not found') return 'Пост не найден.';
  if (normalized === 'comment not found') return 'Комментарий не найден.';
  if (normalized === 'parent comment not found') return 'Родительский комментарий не найден.';
  if (normalized === 'comment deleted') return 'Комментарий уже удален.';
  if (normalized === 'edit not found') return 'Правка не найдена.';
  if (normalized === 'photo not found') return 'Фотография не найдена.';
  if (normalized === 'source post not found') return 'Исходный пост не найден.';
  if (normalized === 'missing access token') return 'Нужно заново войти в аккаунт.';
  if (normalized === 'invalid code') return 'Указан неверный код.';
  if (normalized === 'code expired') return 'Срок действия кода истек.';
  if (normalized === 'too many attempts') return 'Превышено число попыток ввода кода.';
  if (normalized === 'resend limit reached') return 'Слишком много запросов кода. Попробуйте позже.';
  if (normalized === 'empty post') return 'Нужно добавить текст или фотографии.';
  if (normalized === 'too many photos') return 'Можно прикрепить не более 10 фотографий.';
  if (normalized === 'invalid multipart form') return 'Некорректная форма отправки.';
  if (normalized === 'invalid properties') return 'Некорректные данные объекта.';
  if (normalized === 'invalid image') return 'Не удалось обработать изображение.';
  if (normalized.includes('toxic')) return 'Текст не прошел автоматическую проверку.';

  return rawMessage || fieldDetails || 'Произошла ошибка при обработке запроса.';
};

const api = axios.create({
  baseURL: '/api/v1',
  withCredentials: true, // For HttpOnly cookies
});

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      try {
        await axios.post('/api/v1/auth/refresh', {}, { withCredentials: true });
        return api(originalRequest);
      } catch (refreshError) {
        // Redirect to login or clear store
        return Promise.reject(refreshError);
      }
    }
    const data = error.response?.data;
    if (data && typeof data === 'object') {
      const rawMessage = typeof data.message === 'string' ? data.message : '';
      const translated = translateApiMessage(rawMessage, data.fields);
      data.raw_message = rawMessage;
      data.message = translated;
      error.message = translated;
    }
    return Promise.reject(error);
  }
);

export default api;

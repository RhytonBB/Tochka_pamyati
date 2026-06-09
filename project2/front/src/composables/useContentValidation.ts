import { reactive } from 'vue';

type ValidationResponse = {
  requires_ack?: boolean;
  reasons?: string[];
  fields?: Record<string, string>;
  high_risk?: boolean;
  duplicates?: Array<{ id: string; name: string; dist: number }>;
};

export function buildTextFingerprint(value: string) {
  return value.trim();
}

export function buildFilesFingerprint(files: File[]) {
  return files
    .map((file) => `${file.name}:${file.size}:${file.lastModified}`)
    .join('|');
}

export function createValidationState() {
  return reactive({
    warnings: [] as string[],
    badFields: {} as Record<string, string>,
    badPhotos: [] as number[],
    badExistingPhotoIds: [] as string[],
    requiresAck: false,
    isDirtyAfterValidation: false,
    isValidating: false,
    lastValidatedFingerprint: '',
    duplicates: [] as Array<{ id: string; name: string; dist: number }>,
  });
}

export function applyValidationResult(
  state: ReturnType<typeof createValidationState>,
  result: ValidationResponse,
  warningMap: Record<string, string> = {}
) {
  state.warnings = [];
  state.badPhotos = [];
  state.badExistingPhotoIds = [];
  state.duplicates = result.duplicates || [];
  state.requiresAck = !!result.requires_ack;
  Object.keys(state.badFields).forEach((key) => delete state.badFields[key]);

  for (const reason of result.reasons || []) {
    if (reason === 'invalid_input') continue;
    state.warnings.push(warningMap[reason] || reason);
  }

  for (const [key, value] of Object.entries(result.fields || {})) {
    state.badFields[key] = String(value);
    if (key.startsWith('photos.')) {
      const idx = Number.parseInt(key.split('.')[1], 10);
      if (!Number.isNaN(idx)) state.badPhotos.push(idx);
    }
    if (key.startsWith('existing_photos.')) {
      state.badExistingPhotoIds.push(key.split('.')[1]);
    }
  }

  if (state.duplicates.length > 0) {
    state.warnings.push('Обнаружены похожие памятники рядом с указанной точкой');
  }
}

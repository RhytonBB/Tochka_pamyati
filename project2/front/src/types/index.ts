export interface UserSanction {
  id: string;
  user_id: string;
  kind: string;
  source: string;
  reason_code: string;
  reason_text?: string;
  scopes: string[];
  starts_at: string;
  ends_at?: string | null;
  status: string;
  created_by?: string | null;
  related_entity_type?: string;
  related_entity_id?: string | null;
  meta?: Record<string, any>;
  created_at: string;
}

export interface RestrictionSummary {
  status: 'active' | 'restricted' | 'login_banned';
  scopes: string[];
  message: string;
  active_sanctions: UserSanction[];
  ends_at?: string | null;
}

export interface TrustEvent {
  id: string;
  delta: number;
  reason_code: string;
  source_type: string;
  source_id?: string | null;
  comment?: string;
  created_at: string;
}

export interface TrustSummary {
  score: number;
  level: 'trusted' | 'standard' | 'risky' | 'restricted';
  label: string;
  message: string;
  restrictions: string[];
  min_score: number;
  max_score: number;
  next_level_label?: string;
  next_level_score?: number | null;
  recent_events: TrustEvent[];
}

export interface User {
  id: string;
  username: string;
  email: string;
  role_id: string;
  role_name?: string; // We'll populate this from repo.Roles if needed, or backend can include it
  trust_score: number;
  city?: string;
  region?: string;
  notification_settings: Record<string, any>;
  is_active: boolean;
  is_blocked: boolean;
  active_sanctions?: UserSanction[];
  restriction_summary?: RestrictionSummary | null;
  trust_summary?: TrustSummary | null;
  created_at: string;
  last_login?: string;
}

export interface Monument {
  id: string;
  name: string;
  lon: number;
  lat: number;
  status: 'pending' | 'approved' | 'rejected';
  author_id: string;
  created_at: string;
  properties: Record<string, any>;
  thumbnail?: string;
  photos?: Photo[];
  region?: string;
  moderation_comment?: string;
}

export interface Post {
  id: string;
  monument_id: string;
  author_id: string;
  author_name?: string;
  monument_name?: string;
  description: string;
  status: 'pending' | 'approved' | 'rejected';
  created_at: string;
  photos: Photo[];
  thumbnail?: string;
  moderation_comment?: string;
  high_risk?: boolean;
  ai_flags?: Record<string, any>;
}

export interface Photo {
  id: string;
  file_path: string;
  thumbnail_path: string;
  preview_path: string;
  exif_data: Record<string, any>;
}

export interface Signal {
  id: string;
  monument_id?: string;
  monument_name?: string;
  lon?: number;
  lat?: number;
  region?: string;
  author_id?: string;
  status: 'pending' | 'confirmed' | 'resolved' | 'rejected';
  urgency: 'low' | 'medium' | 'high';
  signal_type: string;
  description: string;
  created_at: string;
  support_count?: number;
  is_supported?: boolean;
  official_response?: string;
  author_name?: string;
  resolution_kind?: 'successful' | 'partial' | 'unsuccessful';
  resolution_comment?: string;
  photos?: Photo[];
  thumbnail?: string;
}

export interface NotificationItem {
  id: string;
  type: string;
  title: string;
  content: string;
  link?: string;
  is_read: boolean;
  created_at: string;
}

export interface SignalComment {
  id: string;
  signal_id: string;
  author_id: string;
  author_name: string;
  parent_id?: string;
  content: string;
  is_hidden: boolean;
  created_at: string;
  edited_at?: string | null;
  deleted_at?: string | null;
  toxic_score?: number;
}

export interface SignalDetail {
  signal: Signal;
  photos: Photo[];
  comments: SignalComment[];
  monument?: Monument;
}

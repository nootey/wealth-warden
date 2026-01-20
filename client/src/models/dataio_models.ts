type TxnSample = {
  transaction_type: string;
  amount: string;
  currency: string;
  category: string;
  description: string;
};

export type CustomImportValidationResponse = {
  count: number;
  filtered_count: number;
  sample: TxnSample;
  step: string;
  categories: string[];
};

export type Import = {
  id?: number;
  user_id: number;
  account_id: number;
  name: string;
  status: string;
  type: string;
  sub_type: string;
  currency: string;
  investments_transferred: boolean;
  repayments_transferred: boolean;
  savings_transferred: boolean;
  started_at: Date;
  completed_at: Date | null;
};

export type Export = {
  user_id: number;
  account_id: number;
  name: string;
  status: string;
  export_type: string;
  currency: string;
  started_at: Date;
  completed_at: Date | null;
};

export interface BackupInfo {
  name: string;
  metadata: BackupMetadata;
}

export interface BackupMetadata {
  app_version: string;
  commit_sha: string;
  build_time: string;
  db_version: number;
  created_at: string;
  backup_size: number;
}

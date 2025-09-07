export type UserSettings = {
    language: string,
    timezone: string,
    theme: string,
    accent: string,
}

export type GeneralSettings = {
    default_locale: string,
    default_timezone: string,
    support_email: string,
    allow_signups: boolean,
    max_user_accounts: number,
}
const toastHelper = {
    formatSuccessToast(title: string, msg: string) {
        let message = {
            'data': {
                'messages': {
                    'success': [] as string[]
                },
                'title': {}
            }
        }
        message['data']['title'] = title;
        message['data']['messages']['success'].push(msg);
        return message;
    },
    formatInfoToast(title: string, msg: string) {
        let message = {
            'data': {
                'messages': {
                    'info': [] as string[]
                },
                'title': {}
            }
        }
        message['data']['title'] = title;
        message['data']['messages']['info'].push(msg);
        return message;
    },
    formatErrorToast(title: string, msg: string){
        let message = {
            'response': {
                'data': {
                    'messages': {
                        'error': [] as string[]
                    },
                    'title': {}
                }
            }
        }
        message['response']['data']['title'] = title;
        message['response']['data']['messages']['error'].push(msg);
        return message;
    },
};

export default toastHelper;
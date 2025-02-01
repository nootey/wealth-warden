interface ValidationObject {
    $error: boolean;
}

const vueHelper = {
    getValidationClass: (state: ValidationObject | null | undefined, errorClass: string) => {
        return {
            [errorClass]: !!state?.$error,
        }
    },
};

export default vueHelper;
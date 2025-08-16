const chartHelper = {
    extractAllBut: (data: any[], index_key: string, key: string) => {

        let totalSum = 0;
        let averageSum = 0;

        data
            .filter(item => item[index_key] !== "Total" && item[index_key] !== key)
            .forEach(item => {
                if (typeof item["total"] === "number") {
                    totalSum += item["total"];
                }
                if (typeof item["average"] === "number") {
                    averageSum += item["average"];
                }
            });

        return { totalSum, averageSum };
    },
    extractAllFor: (data: any[], index_key: string, key: string) => {

        let totalSum = 0;
        let averageSum = 0;

        data
            .filter(item => item[index_key] !== "Total" && item[index_key] === key)
            .forEach(item => {
                if (typeof item["total"] === "number") {
                    totalSum += item["total"];
                }
                if (typeof item["average"] === "number") {
                    averageSum += item["average"];
                }
            });

        return { totalSum, averageSum };
    }
};

export default chartHelper;

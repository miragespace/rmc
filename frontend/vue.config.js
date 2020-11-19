module.exports = {
    chainWebpack: config => {
        config
            .plugin('html')
            .tap(args => {
                args[0].title = "Rent a Minecraft Server"
                return args
            })
    }
}
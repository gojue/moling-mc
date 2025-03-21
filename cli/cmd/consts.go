package cmd

const (
	CliName            = "moling"
	CliNameZh          = "魔灵"
	CliDescription     = "A local office automation magic assistant, Moling can help you perform file viewing, reading, and other related tasks, as well as assist with command-line operations.\n\n"
	CliDescriptionZh   = "本地办公自动化魔法助手，可以帮你实现文件查看、阅读等操作，也可以帮你执行命令行操作。"
	CliHomepage        = "https://gojue.cc/moling"
	CliAuthor          = "CFC4N <cfc4ncs@gmail.com>"
	CliGithubRepo      = "https://github.com/gojue/moling"
	CliDescriptionLong = `
Moling (魔灵) is a computer-based MCP Server that implements system interaction through operating system APIs, enabling file system operations such as reading, writing, merging, statistics, and aggregation, as well as the ability to execute system commands. It is a dependency-free local office automation assistant.

Requiring no installation of any dependencies, Moling can be run directly and is compatible with multiple operating systems, including Windows, Linux, and MacOS. This eliminates the hassle of dealing with environment conflicts involving Node.js, Python, and other development environments.

Usage:
  moling --allow-dir="/tmp,/home/user"
  moling -h
`
	CliDescriptionLongZh = `Moling(魔灵) 是一个computer-use的MCP Server，基于操作系统API实现了系统交互，可以实现文件系统的读写、合并、统计、聚合等操作，也可以执行系统命令操作。是一个无需任何依赖的本地办公自动化助手。
没有任何安装依赖，直接运行，兼容Windows、Linux、MacOS等操作系统。再也不用苦恼NodeJS、Python等环境冲突等问题了。

Usage:
  moling --allow-dir="/tmp,/home/user"
  moling -h
`
)

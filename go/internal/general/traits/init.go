// Package traits 通过 init() 机制自动注册所有特性
//
// 添加新特性：
//   1. 在本包新建文件，实现 general.Trait 接口
//   2. 在该文件 init() 里调用 general.Register(&MyTrait{})
//   3. 在使用方 import "hero3/internal/general/traits" 触发包初始化
package traits

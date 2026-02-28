// 类型定义
export interface FormInstance {
  model: Record<string, any>
  rules?: FormRules
  validateCallback: ((valid: boolean) => void) | null
  validate: (callback: (valid: boolean) => void) => void
  resetFields: () => void
  clearValidate: () => void
}

export type ValidateCallback = (valid: boolean) => void

export interface FormRules {
  [key: string]: FormRule[]
}

export interface FormRule {
  required?: boolean
  message?: string
  trigger?: 'blur' | 'change'
  min?: number
  max?: number
  pattern?: RegExp
  validator?: (value: any, callback: (error?: Error) => void) => void
}

export interface FormItemInstance {
  validate: () => Promise<boolean>
  resetField: () => void
  clearValidate: () => void
}


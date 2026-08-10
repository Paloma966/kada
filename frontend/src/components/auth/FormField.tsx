interface FormFieldProps {
  id: string;
  label: string;
  error?: string;
  children: React.ReactNode;
}

export function FormField({ id, label, error, children }: FormFieldProps) {
  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-neutral-100">
        {label}
      </label>
      {children}
      {error && <p className="mt-1.5 text-xs text-red-400">{error}</p>}
    </div>
  );
}

/** 输入框基础样式（不含边框色/聚焦态）——暗色星空主题 */
export const inputBase =
  "w-full rounded-lg border bg-white/10 px-4 py-2.5 text-sm text-white placeholder:text-neutral-400/70 transition focus:outline-none focus:ring-2";

/** 输入框边框/聚焦状态：invalid 传 true 显示红色错误态 */
export function fieldState(invalid: boolean): string {
  return invalid
    ? "border-red-400/60 focus:border-red-400 focus:ring-red-400/30"
    : "border-white/15 focus:border-indigo-400 focus:ring-indigo-400/30";
}

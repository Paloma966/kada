interface FormFieldProps {
  id: string;
  label: string;
  error?: string;
  children: React.ReactNode;
}

export function FormField({ id, label, error, children }: FormFieldProps) {
  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-neutral-700">
        {label}
      </label>
      {children}
      {error && <p className="mt-1.5 text-xs text-red-600">{error}</p>}
    </div>
  );
}

/** 输入框基础样式（不含边框色/聚焦态） */
export const inputBase =
  "w-full rounded-lg border bg-white px-4 py-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 transition focus:outline-none focus:ring-2";

/** 输入框边框/聚焦状态：invalid 传 true 显示红色错误态 */
export function fieldState(invalid: boolean): string {
  return invalid
    ? "border-red-300 focus:border-red-300 focus:ring-red-100"
    : "border-neutral-200 focus:border-indigo-500 focus:ring-indigo-100";
}

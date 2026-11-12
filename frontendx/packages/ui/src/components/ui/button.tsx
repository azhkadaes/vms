// shadcn/ui source-owned primitive; CSS adapted to the app's existing tokens.
// https://ui.shadcn.com/r/styles/new-york-v4/button.json (MIT; LICENSE.shadcn)
import { forwardRef, type ButtonHTMLAttributes } from "react";
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "default" | "outline" | "ghost" | "secondary";
  size?: "default" | "icon" | "sm";
};
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button({ className = "", variant = "outline", size = "default", type = "button", ...props }, ref) {
    return <button ref={ref} type={type} data-slot="button"
      className={`shadcn-button shadcn-button-${variant} shadcn-button-size-${size} ${className}`} {...props} />;
  },
);

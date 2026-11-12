// Adapted from shadcn/ui's Radix Popover (MIT; LICENSE.shadcn).
// https://ui.shadcn.com/r/styles/new-york-v4/popover.json
import { forwardRef, type ComponentPropsWithoutRef } from "react";
import * as Primitive from "@radix-ui/react-popover";
export const Popover = Primitive.Root;
export const PopoverTrigger = Primitive.Trigger;
export const PopoverClose = Primitive.Close;
type PopoverContentProps = ComponentPropsWithoutRef<typeof Primitive.Content> & { container?: HTMLElement };
export const PopoverContent = forwardRef<HTMLDivElement, PopoverContentProps>(
  function PopoverContent({ className = "", align = "start", sideOffset = 6, container, ...props }, ref) {
    return <Primitive.Portal container={container}>
      <Primitive.Content ref={ref} data-slot="popover-content" align={align}
        sideOffset={sideOffset} collisionPadding={12}
        className={`shadcn-popover ${className}`} {...props} />
    </Primitive.Portal>;
  },
);

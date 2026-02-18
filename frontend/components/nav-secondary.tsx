"use client" // Mark as client component, can use browser APIs and interactive features

// Import React library
import * as React from "react"
// Import icon type
import { type Icon } from "@/components/icons"

// Import sidebar related components
import {
  SidebarGroup,        // Sidebar group
  SidebarGroupContent, // Sidebar group content
  SidebarMenu,         // Sidebar menu
  SidebarMenuButton,   // Sidebar menu button
  SidebarMenuItem,     // Sidebar menu item
} from '@/components/ui/sidebar'

/**
 * Secondary navigation component
 * Displays secondary navigation menu items, typically used for settings, help, etc.
 * 
 * @param {Object} props - Component properties
 * @param {Array} props.items - Navigation items array
 * @param {string} props.items[].title - Navigation item title
 * @param {string} props.items[].url - Navigation item link
 * @param {Icon} props.items[].icon - Navigation item icon
 * @param {...any} props - Other properties passed to SidebarGroup
 */
export function NavSecondary({
  items,
  ...props  // Other properties passed to SidebarGroup
}: {
  items: {
    title: string  // Navigation item title
    url: string    // Navigation item URL
    icon: Icon     // Navigation item icon
  }[]
} & React.ComponentPropsWithoutRef<typeof SidebarGroup>) {
  return (
    <SidebarGroup {...props}>  {/* Pass all other properties */}
      {/* Sidebar group content */}
      <SidebarGroupContent>
        {/* Sidebar menu */}
        <SidebarMenu>
          {/* Iterate through secondary navigation items */}
          {items.map((item) => (
            <SidebarMenuItem key={item.title}>
              {/* Navigation menu button, rendered as link using asChild */}
              <SidebarMenuButton asChild>
                <a href={item.url}>              {/* Navigation link */}
                  <item.icon />                   {/* Navigation item icon */}
                  <span>{item.title}</span>       {/* Navigation item title */}
                </a>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}

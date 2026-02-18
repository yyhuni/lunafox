"use client" // Mark as client component, can use browser APIs and interactive features

import React from "react"
// Import icon components
import {
  IconDotsVertical,  // Vertical three-dot icon
  IconKey,           // Key icon
  IconLogout,        // Logout icon
} from "@/components/icons"

// Import avatar related components
import {
  Avatar,        // Avatar container
  AvatarFallback, // Avatar fallback display
  AvatarImage,   // Avatar image
} from '@/components/ui/avatar'
// Import dropdown menu related components
import {
  DropdownMenu,          // Dropdown menu container
  DropdownMenuContent,   // Dropdown menu content
  DropdownMenuItem,      // Dropdown menu item
  DropdownMenuLabel,     // Dropdown menu label
  DropdownMenuSeparator, // Dropdown menu separator
  DropdownMenuTrigger,   // Dropdown menu trigger
} from '@/components/ui/dropdown-menu'
// Import sidebar related components
import {
  SidebarMenu,       // Sidebar menu
  SidebarMenuButton, // Sidebar menu button
  SidebarMenuItem,   // Sidebar menu item
  useSidebar,        // Sidebar Hook
} from '@/components/ui/sidebar'
import { useAuth, useLogout } from '@/hooks/use-auth'
import { ChangePasswordDialog } from '@/components/auth/change-password-dialog'

/**
 * User navigation component
 * Displays user information and user-related action menu
 * 
 * @param {Object} props - Component properties
 * @param {Object} props.user - User information
 * @param {string} props.user.name - User name
 * @param {string} props.user.email - User email
 * @param {string} props.user.avatar - User avatar URL
 */
export function NavUser({
  user,
}: {
  user: {
    name: string   // User name
    email: string  // User email
    avatar: string // User avatar URL
  }
}) {
  const { isMobile } = useSidebar() // Get mobile state
  const { data: auth } = useAuth()
  const { mutate: logout, isPending: isLoggingOut } = useLogout()
  const [showChangePassword, setShowChangePassword] = React.useState(false)
  
  // Use real username (if logged in)
  const displayName = auth?.user?.username || user.name

  return (
    <>
    <ChangePasswordDialog 
      open={showChangePassword} 
      onOpenChange={setShowChangePassword} 
    />
    <SidebarMenu>
      <SidebarMenuItem>
        {/* User dropdown menu */}
        <DropdownMenu>
          {/* Dropdown menu trigger */}
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"                                                    // Large size
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground" // Style when open
            >
              {/* User avatar */}
              <Avatar className="h-8 w-8 rounded-lg grayscale">         {/* 8x8 size, rounded, grayscale */}
                <AvatarImage src={user.avatar} alt={user.name} />       {/* User avatar image */}
                <AvatarFallback className="rounded-lg">CN</AvatarFallback> {/* Fallback display */}
              </Avatar>
              {/* User information area */}
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{displayName}</span>  {/* User name */}
                <span className="text-muted-foreground truncate text-xs">  {/* User email */}
                  {/* {user.email} */}
                </span>
              </div>
              {/* Three-dot menu icon */}
              <IconDotsVertical className="ml-auto size-4" />           {/* Auto left margin, 4x4 size */}
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          {/* Dropdown menu content */}
          <DropdownMenuContent
            className="rounded-lg"                                     // Rounded corners
            side={isMobile ? "bottom" : "right"}                        // Bottom on mobile, right on desktop
            align="end"                                                 // End alignment
            sideOffset={4}                                             // 4px offset
          >
            {/* User information label */}
            <DropdownMenuLabel className="p-0 font-normal">           {/* No padding, normal font */}
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                {/* User avatar */}
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarImage src={user.avatar} alt={user.name} />     {/* User avatar image */}
                  <AvatarFallback className="rounded-lg">CN</AvatarFallback> {/* Fallback display */}
                </Avatar>
                {/* User information */}
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{displayName}</span>  {/* User name */}
                  <span className="text-muted-foreground truncate text-xs">  {/* User email */}
                    {/* {user.email} */}
                  </span>
                </div>
              </div>
            </DropdownMenuLabel>
            {/* Separator */}
            <DropdownMenuSeparator />
            {/* Change password */}
            <DropdownMenuItem onClick={() => setShowChangePassword(true)}>
              <IconKey />
              Change Password
            </DropdownMenuItem>
            {/* Logout option */}
            <DropdownMenuItem 
              onClick={() => logout()}
              disabled={isLoggingOut}
            >
              <IconLogout />
              {isLoggingOut ? 'Logging out…' : 'Logout'}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
    </>
  )
}

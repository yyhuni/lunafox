"use client" // Mark as client component, can use browser APIs and interactive features

import React from "react"
import {
  IconKey,           // Key icon
  IconLogout,        // Logout icon
} from "@/components/icons"

// Import dropdown menu related components
import {
  DropdownMenuItem,      // Dropdown menu item
} from '@/components/ui/dropdown-menu'
import { LunaFoxMark } from "@/components/brand/lunafox-mark"
import { SidebarUserMenu, SidebarUserMenuSkeleton } from '@/components/shared/dropdown-menu-owners'
// Import sidebar related components
import {
  SidebarMenu,       // Sidebar menu
  SidebarMenuItem,   // Sidebar menu item
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
    avatar?: string // User avatar URL
  }
}) {
  const { data: auth } = useAuth()
  const { mutate: logout, isPending: isLoggingOut } = useLogout()
  const [showChangePassword, setShowChangePassword] = React.useState(false)
  
  // Use real username (if logged in)
  const displayName = auth?.user?.username || user.name
  const displaySubline = auth?.user?.email || user.email

  return (
    <>
    <ChangePasswordDialog 
      open={showChangePassword} 
      onOpenChange={setShowChangePassword} 
    />
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarUserMenu
          userName={displayName}
          userSubline={displaySubline}
          avatarSrc={user.avatar}
          avatarAlt={displayName}
          avatarFallback={<LunaFoxMark className="size-5" decorative />}
        >
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
        </SidebarUserMenu>
      </SidebarMenuItem>
    </SidebarMenu>
    </>
  )
}

export function NavUserSkeleton() {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarUserMenuSkeleton />
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

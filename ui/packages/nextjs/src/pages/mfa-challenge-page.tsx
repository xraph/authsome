"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { MFAChallengeFormStyled } from "@authsome/ui-components";

export interface MFAChallengePageProps {
  enrollmentId: string;
  afterVerifyUrl?: string;
  logo?: React.ReactNode;
}

export function MFAChallengePage({
  enrollmentId,
  afterVerifyUrl = "/",
  logo,
}: MFAChallengePageProps) {
  const router = useRouter();

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <MFAChallengeFormStyled
        enrollmentId={enrollmentId}
        onSuccess={() => router.push(afterVerifyUrl)}
        logo={logo}
      />
    </div>
  );
}

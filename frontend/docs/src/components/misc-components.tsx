import React from 'react';

// 1. Gradient Geometric Background
export function SvgGradientBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1342" height="1199" fill="none" className="absolute top-0 right-0 -z-10 origin-right scale-30 md:scale-50 lg:scale-100">
            <path fill="#D9D9D9" d="M914.912 1197.77 747.793 808.811l115.698-221.478 334.239 73.826 109.08 196.135-391.898 340.476Z"></path>
            <path fill="url(#a)" d="M914.912 1197.77 747.793 808.811l115.698-221.478 334.239 73.826 109.08 196.135-391.898 340.476Z"></path>
            <path stroke="url(#b)" strokeWidth="0.631" d="M914.912 1197.77 747.793 808.811l115.698-221.478 334.239 73.826 109.08 196.135-391.898 340.476Z"></path>
            <path fill="url(#c)" d="m875.715 420.318 203.405-357.96c50.52-10.487-50.57 96.246 0 186.332 80.45 143.304 298.36 312.903 256.86 419.243-67.58 173.19-306.7 49.523-396.529 0-71.863-39.618-72.434-181.585-63.736-247.615Z"></path>
            <path fill="url(#d)" d="m46.623 746.37 908.336-619.388 130.381-66.714-46.89 196.709-156.685 413.622c-27.829 50.066-111.545 120.16-223.775 0-98.592-105.557-466.882-3.975-611.367 75.771L.814 777.607c10.115-9.59 25.82-20.205 45.809-31.237Z"></path>
            <g filter="url(#e)">
                <path fill="url(#f)" d="m883.093 595.649 164.727-565.43 4.66 326.52-169.387 238.91Z"></path>
            </g>
            <defs>
                <linearGradient id="a" x1="1027.3" x2="1027.73" y1="587.333" y2="1198.11" gradientUnits="userSpaceOnUse">
                    <stop offset="0" stopColor="#9D83E7"></stop>
                    <stop offset="0.516" stopColor="#D445E7"></stop>
                </linearGradient>
                <linearGradient id="b" x1="1027.3" x2="1027.3" y1="587.333" y2="1197.77" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#10CBF4"></stop>
                    <stop offset="1" stopColor="#10CBF4" stopOpacity="0"></stop>
                </linearGradient>
                <linearGradient id="c" x1="871.897" x2="1188.44" y1="575.509" y2="575.628" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#9259ED"></stop>
                    <stop offset="0.514" stopColor="#CF54EE"></stop>
                    <stop offset="1" stopColor="#FB8684"></stop>
                </linearGradient>
                <linearGradient id="d" x1="676.669" x2="677.051" y1="60.268" y2="757.516" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#B956EE"></stop>
                    <stop offset="1" stopColor="#9672FF"></stop>
                </linearGradient>
                <linearGradient id="f" x1="1020.81" x2="814.267" y1="202.771" y2="477.618" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#FB07FF"></stop>
                    <stop offset="0.505" stopColor="#FF6847"></stop>
                    <stop offset="1" stopColor="#FF474A"></stop>
                </linearGradient>
                <filter id="e" width="228.968" height="625.009" x="853.303" y="0.429" colorInterpolationFilters="sRGB" filterUnits="userSpaceOnUse">
                    <feFlood floodOpacity="0" result="BackgroundImageFix"></feFlood>
                    <feBlend in="SourceGraphic" in2="BackgroundImageFix" result="shape"></feBlend>
                    <feGaussianBlur result="effect1_foregroundBlur_401_39842" stdDeviation="14.895"></feGaussianBlur>
                </filter>
            </defs>
        </svg>
    );
}

// 2. Wave Pattern Background
export function SvgWaveBackground({ colors = ['#4F46E5', '#7C3AED', '#EC4899'] }: { colors?: string[] }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1440" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <path fill="url(#wave-a)" d="M0 400 Q360 200 720 400 T1440 400 V800 H0 Z" opacity="0.3"></path>
            <path fill="url(#wave-b)" d="M0 500 Q360 350 720 500 T1440 500 V800 H0 Z" opacity="0.5"></path>
            <path fill="url(#wave-c)" d="M0 600 Q360 500 720 600 T1440 600 V800 H0 Z"></path>
            <defs>
                <linearGradient id="wave-a" x1="0" x2="1440" y1="400" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor={colors[0]}></stop>
                    <stop offset="1" stopColor={colors[1]}></stop>
                </linearGradient>
                <linearGradient id="wave-b" x1="0" x2="1440" y1="500" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor={colors[1]}></stop>
                    <stop offset="1" stopColor={colors[2]}></stop>
                </linearGradient>
                <linearGradient id="wave-c" x1="0" x2="1440" y1="600" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor={colors[2]}></stop>
                    <stop offset="1" stopColor={colors[0]}></stop>
                </linearGradient>
            </defs>
        </svg>
    );
}

// 3. Circular Gradient Burst
export function SvgCircleBurstBackground({ primaryColor = '#FF6B6B', secondaryColor = '#4ECDC4' }: { primaryColor?: string; secondaryColor?: string }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="1200" fill="none" className="absolute top-0 left-0 -z-10 opacity-60">
            <circle cx="600" cy="600" r="500" fill="url(#burst-a)"></circle>
            <circle cx="800" cy="400" r="300" fill="url(#burst-b)" opacity="0.7"></circle>
            <circle cx="400" cy="800" r="350" fill="url(#burst-c)" opacity="0.5"></circle>
            <defs>
                <radialGradient id="burst-a" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(600 600) scale(500)">
                    <stop stopColor={primaryColor}></stop>
                    <stop offset="1" stopColor={primaryColor} stopOpacity="0"></stop>
                </radialGradient>
                <radialGradient id="burst-b" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(800 400) scale(300)">
                    <stop stopColor={secondaryColor}></stop>
                    <stop offset="1" stopColor={secondaryColor} stopOpacity="0"></stop>
                </radialGradient>
                <radialGradient id="burst-c" cx="0" cy="0" r="1" gradientUnits="userSpaceOnUse" gradientTransform="translate(400 800) scale(350)">
                    <stop stopColor="#F7B731"></stop>
                    <stop offset="1" stopColor="#F7B731" stopOpacity="0"></stop>
                </radialGradient>
            </defs>
        </svg>
    );
}

// 4. Angular Polygon Background
export function SvgPolygonBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1400" height="1000" fill="none" className="absolute inset-0 -z-10">
            <polygon points="700,100 1200,400 1000,900 400,900 200,400" fill="url(#poly-a)" opacity="0.6"></polygon>
            <polygon points="900,200 1300,500 1100,800 500,800 300,500" fill="url(#poly-b)" opacity="0.4"></polygon>
            <polygon points="500,300 800,200 1100,500 800,800 500,700" fill="url(#poly-c)" opacity="0.5"></polygon>
            <defs>
                <linearGradient id="poly-a" x1="200" x2="1200" y1="100" y2="900" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#667EEA"></stop>
                    <stop offset="1" stopColor="#764BA2"></stop>
                </linearGradient>
                <linearGradient id="poly-b" x1="300" x2="1300" y1="200" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#F093FB"></stop>
                    <stop offset="1" stopColor="#F5576C"></stop>
                </linearGradient>
                <linearGradient id="poly-c" x1="500" x2="1100" y1="200" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#4FACFE"></stop>
                    <stop offset="1" stopColor="#00F2FE"></stop>
                </linearGradient>
            </defs>
        </svg>
    );
}

// 5. Mesh Gradient Background
export function SvgMeshBackground({ blur = 40 }: { blur?: number }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <g filter="url(#mesh-blur)">
                <ellipse cx="300" cy="200" rx="200" ry="150" fill="#FF6B9D"></ellipse>
                <ellipse cx="800" cy="300" rx="250" ry="200" fill="#C44569"></ellipse>
                <ellipse cx="600" cy="600" rx="300" ry="250" fill="#FFA94D"></ellipse>
                <ellipse cx="200" cy="700" rx="200" ry="180" fill="#A770EF"></ellipse>
            </g>
            <defs>
                <filter id="mesh-blur" x="-50%" y="-50%" width="200%" height="200%">
                    <feGaussianBlur in="SourceGraphic" stdDeviation={blur}></feGaussianBlur>
                </filter>
            </defs>
        </svg>
    );
}

// 6. Striped Diagonal Background
export function SvgStripedBackground({ stripeColor = '#6366F1', bgColor = '#818CF8', opacity = 0.3 }: { stripeColor?: string; bgColor?: string; opacity?: number }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <rect width="1200" height="800" fill={bgColor}></rect>
            <g opacity={opacity}>
                <path d="M0 0 L200 0 L0 200 Z" fill={stripeColor}></path>
                <path d="M200 0 L400 0 L0 400 L0 200 Z" fill={stripeColor}></path>
                <path d="M400 0 L600 0 L0 600 L0 400 Z" fill={stripeColor}></path>
                <path d="M600 0 L800 0 L0 800 L0 600 Z" fill={stripeColor}></path>
                <path d="M800 0 L1000 0 L200 800 L0 800 Z" fill={stripeColor}></path>
                <path d="M1000 0 L1200 0 L400 800 L200 800 Z" fill={stripeColor}></path>
                <path d="M1200 0 L1200 200 L600 800 L400 800 Z" fill={stripeColor}></path>
                <path d="M1200 200 L1200 400 L800 800 L600 800 Z" fill={stripeColor}></path>
                <path d="M1200 400 L1200 600 L1000 800 L800 800 Z" fill={stripeColor}></path>
                <path d="M1200 600 L1200 800 L1000 800 Z" fill={stripeColor}></path>
            </g>
        </svg>
    );
}

// 7. Hexagon Pattern Background
export function SvgHexagonBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full" opacity="0.4">
            <defs>
                <pattern id="hexagons" x="0" y="0" width="100" height="86.6" patternUnits="userSpaceOnUse">
                    <polygon points="50,0 93.3,25 93.3,75 50,100 6.7,75 6.7,25" fill="none" stroke="url(#hex-gradient)" strokeWidth="2"></polygon>
                </pattern>
                <linearGradient id="hex-gradient" x1="0" x2="100" y1="0" y2="100" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#3B82F6"></stop>
                    <stop offset="1" stopColor="#8B5CF6"></stop>
                </linearGradient>
            </defs>
            <rect width="1200" height="800" fill="url(#hexagons)"></rect>
        </svg>
    );
}

// 8. Organic Blob Background
export function SvgBlobBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <path d="M300,150 Q450,50 600,150 T900,250 Q950,400 850,550 T650,700 Q450,750 300,650 T150,450 Q100,300 300,150 Z" fill="url(#blob-a)" opacity="0.6"></path>
            <path d="M700,100 Q850,50 950,150 T1100,350 Q1150,500 1000,600 T750,650 Q600,600 550,450 T600,250 Q650,150 700,100 Z" fill="url(#blob-b)" opacity="0.5"></path>
            <g filter="url(#blob-blur)">
                <path d="M200,500 Q300,450 400,500 T600,600 Q650,700 500,750 T300,700 Q200,650 200,500 Z" fill="url(#blob-c)" opacity="0.7"></path>
            </g>
            <defs>
                <linearGradient id="blob-a" x1="150" x2="900" y1="150" y2="700" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#FF6B6B"></stop>
                    <stop offset="1" stopColor="#FFE66D"></stop>
                </linearGradient>
                <linearGradient id="blob-b" x1="550" x2="1150" y1="100" y2="650" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#4ECDC4"></stop>
                    <stop offset="1" stopColor="#556270"></stop>
                </linearGradient>
                <radialGradient id="blob-c" cx="400" cy="600" r="200" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#A8E6CF"></stop>
                    <stop offset="1" stopColor="#3D84A8"></stop>
                </radialGradient>
                <filter id="blob-blur">
                    <feGaussianBlur stdDeviation="20"></feGaussianBlur>
                </filter>
            </defs>
        </svg>
    );
}

// 9. Grid Dots Background
export function SvgGridDotsBackground({ dotColor = '#8B5CF6', spacing = 30 }: { dotColor?: string; spacing?: number }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full" opacity="0.3">
            <defs>
                <pattern id="dots" x="0" y="0" width={spacing} height={spacing} patternUnits="userSpaceOnUse">
                    <circle cx={spacing / 2} cy={spacing / 2} r="2" fill={dotColor}></circle>
                </pattern>
            </defs>
            <rect width="1200" height="800" fill="url(#dots)"></rect>
        </svg>
    );
}

// 10. Triangular Mosaic Background
export function SvgTriangleMosaicBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <path d="M0,0 L200,0 L100,173.2 Z" fill="#FF6B9D" opacity="0.7"></path>
            <path d="M200,0 L400,0 L300,173.2 Z" fill="#C44569" opacity="0.6"></path>
            <path d="M400,0 L600,0 L500,173.2 Z" fill="#FFA94D" opacity="0.8"></path>
            <path d="M100,173.2 L300,173.2 L200,346.4 Z" fill="#6C5CE7" opacity="0.7"></path>
            <path d="M300,173.2 L500,173.2 L400,346.4 Z" fill="#A29BFE" opacity="0.6"></path>
            <path d="M600,0 L800,0 L700,173.2 Z" fill="#FD79A8" opacity="0.7"></path>
            <path d="M800,0 L1000,0 L900,173.2 Z" fill="#FDCB6E" opacity="0.6"></path>
            <path d="M500,173.2 L700,173.2 L600,346.4 Z" fill="#00B894" opacity="0.8"></path>
            <path d="M0,346.4 L200,346.4 L100,519.6 Z" fill="#00CEC9" opacity="0.7"></path>
            <path d="M200,346.4 L400,346.4 L300,519.6 Z" fill="#74B9FF" opacity="0.6"></path>
        </svg>
    );
}

// 11. Curved Lines Background
export function SvgCurvedLinesBackground({ lineColor = '#6366F1' }: { lineColor?: string }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <path d="M0,100 Q300,50 600,100 T1200,100" stroke="url(#curve-a)" strokeWidth="3" fill="none" opacity="0.6"></path>
            <path d="M0,200 Q300,150 600,200 T1200,200" stroke="url(#curve-b)" strokeWidth="3" fill="none" opacity="0.5"></path>
            <path d="M0,300 Q300,250 600,300 T1200,300" stroke="url(#curve-c)" strokeWidth="3" fill="none" opacity="0.4"></path>
            <path d="M0,400 Q300,350 600,400 T1200,400" stroke="url(#curve-d)" strokeWidth="3" fill="none" opacity="0.6"></path>
            <path d="M0,500 Q300,450 600,500 T1200,500" stroke="url(#curve-e)" strokeWidth="3" fill="none" opacity="0.5"></path>
            <path d="M0,600 Q300,550 600,600 T1200,600" stroke="url(#curve-f)" strokeWidth="3" fill="none" opacity="0.4"></path>
            <defs>
                <linearGradient id="curve-a" x1="0" x2="1200" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor={lineColor}></stop>
                    <stop offset="1" stopColor="#8B5CF6"></stop>
                </linearGradient>
                <linearGradient id="curve-b" x1="0" x2="1200" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#8B5CF6"></stop>
                    <stop offset="1" stopColor="#EC4899"></stop>
                </linearGradient>
                <linearGradient id="curve-c" x1="0" x2="1200" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#EC4899"></stop>
                    <stop offset="1" stopColor="#F59E0B"></stop>
                </linearGradient>
                <linearGradient id="curve-d" x1="0" x2="1200" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#F59E0B"></stop>
                    <stop offset="1" stopColor="#10B981"></stop>
                </linearGradient>
                <linearGradient id="curve-e" x1="0" x2="1200" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#10B981"></stop>
                    <stop offset="1" stopColor="#3B82F6"></stop>
                </linearGradient>
                <linearGradient id="curve-f" x1="0" x2="1200" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#3B82F6"></stop>
                    <stop offset="1" stopColor="#6366F1"></stop>
                </linearGradient>
            </defs>
        </svg>
    );
}

// 12. Star Field Background
export function SvgStarFieldBackground({ starCount = 50, starColor = '#FFF' }: { starCount?: number; starColor?: string }) {
    const stars = Array.from({ length: starCount }, (_, i) => ({
        cx: Math.random() * 1200,
        cy: Math.random() * 800,
        r: Math.random() * 2 + 0.5,
        opacity: Math.random() * 0.5 + 0.5
    }));

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <rect width="1200" height="800" fill="#0F172A"></rect>
            {stars.map((star, i) => (
                <circle key={i} cx={star.cx} cy={star.cy} r={star.r} fill={starColor} opacity={star.opacity}></circle>
            ))}
        </svg>
    );
}

// 13. Abstract Flow Background
export function SvgAbstractFlowBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1400" height="900" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <g filter="url(#flow-blur)">
                <path d="M-100,450 C200,200 400,600 700,450 S1200,200 1500,450" stroke="url(#flow-a)" strokeWidth="80" fill="none" opacity="0.6"></path>
                <path d="M-100,500 C200,300 400,700 700,500 S1200,300 1500,500" stroke="url(#flow-b)" strokeWidth="60" fill="none" opacity="0.5"></path>
                <path d="M-100,550 C200,400 400,800 700,550 S1200,400 1500,550" stroke="url(#flow-c)" strokeWidth="40" fill="none" opacity="0.4"></path>
            </g>
            <defs>
                <linearGradient id="flow-a" x1="0" x2="1400" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#667EEA"></stop>
                    <stop offset="0.5" stopColor="#764BA2"></stop>
                    <stop offset="1" stopColor="#F093FB"></stop>
                </linearGradient>
                <linearGradient id="flow-b" x1="0" x2="1400" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#4FACFE"></stop>
                    <stop offset="0.5" stopColor="#00F2FE"></stop>
                    <stop offset="1" stopColor="#43E97B"></stop>
                </linearGradient>
                <linearGradient id="flow-c" x1="0" x2="1400" y1="0" y2="0" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#FA709A"></stop>
                    <stop offset="0.5" stopColor="#FEE140"></stop>
                    <stop offset="1" stopColor="#30CFD0"></stop>
                </linearGradient>
                <filter id="flow-blur">
                    <feGaussianBlur stdDeviation="30"></feGaussianBlur>
                </filter>
            </defs>
        </svg>
    );
}

// 14. Radial Burst Background
export function SvgRadialBurstBackground({ segments = 12, colors = ['#FF6B6B', '#4ECDC4', '#FFE66D'] }: { segments?: number; colors?: string[] }) {
    const angleStep = 360 / segments;
    const rays = Array.from({ length: segments }, (_, i) => {
        const angle = i * angleStep;
        const nextAngle = (i + 1) * angleStep;
        const color = colors[i % colors.length];
        return { angle, nextAngle, color };
    });

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="1200" fill="none" className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 -z-10" opacity="0.4">
            {rays.map((ray, i) => (
                <path
                    key={i}
                    d={`M 600 600 L ${600 + Math.cos((ray.angle * Math.PI) / 180) * 800} ${600 + Math.sin((ray.angle * Math.PI) / 180) * 800} A 800 800 0 0 1 ${600 + Math.cos((ray.nextAngle * Math.PI) / 180) * 800} ${600 + Math.sin((ray.nextAngle * Math.PI) / 180) * 800} Z`}
                    fill={ray.color}
                    opacity="0.6"
                ></path>
            ))}
        </svg>
    );
}

// 15. Layered Shapes Background
export function SvgLayeredShapesBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <rect x="100" y="100" width="400" height="300" rx="20" fill="url(#layer-a)" opacity="0.4"></rect>
            <circle cx="900" cy="200" r="150" fill="url(#layer-b)" opacity="0.5"></circle>
            <polygon points="300,600 500,500 700,600 600,750 400,750" fill="url(#layer-c)" opacity="0.6"></polygon>
            <ellipse cx="1000" cy="600" rx="180" ry="120" fill="url(#layer-d)" opacity="0.4"></ellipse>
            <rect x="50" y="500" width="250" height="250" rx="125" fill="url(#layer-e)" opacity="0.5"></rect>
            <defs>
                <linearGradient id="layer-a" x1="100" x2="500" y1="100" y2="400" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#667EEA"></stop>
                    <stop offset="1" stopColor="#764BA2"></stop>
                </linearGradient>
                <radialGradient id="layer-b" cx="900" cy="200" r="150" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#F093FB"></stop>
                    <stop offset="1" stopColor="#F5576C"></stop>
                </radialGradient>
                <linearGradient id="layer-c" x1="300" x2="700" y1="500" y2="750" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#4FACFE"></stop>
                    <stop offset="1" stopColor="#00F2FE"></stop>
                </linearGradient>
                <radialGradient id="layer-d" cx="1000" cy="600" r="180" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#43E97B"></stop>
                    <stop offset="1" stopColor="#38F9D7"></stop>
                </radialGradient>
                <linearGradient id="layer-e" x1="50" x2="300" y1="500" y2="750" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#FA709A"></stop>
                    <stop offset="1" stopColor="#FEE140"></stop>
                </linearGradient>
            </defs>
        </svg>
    );
}

// 16. Diagonal Lines Pattern
export function SvgDiagonalLinesBackground({ lineSpacing = 20, lineWidth = 2, lineColor = '#8B5CF6' }: { lineSpacing?: number; lineWidth?: number; lineColor?: string }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full" opacity="0.2">
            <defs>
                <pattern id="diagonals" x="0" y="0" width={lineSpacing} height={lineSpacing} patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
                    <line x1="0" y1="0" x2="0" y2={lineSpacing} stroke={lineColor} strokeWidth={lineWidth}></line>
                </pattern>
            </defs>
            <rect width="1200" height="800" fill="url(#diagonals)"></rect>
        </svg>
    );
}

// 17. Cellular Network Background
export function SvgCellularBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <g stroke="url(#cell-gradient)" strokeWidth="2" opacity="0.4">
                <line x1="100" y1="100" x2="300" y2="200"></line>
                <line x1="300" y1="200" x2="500" y2="150"></line>
                <line x1="500" y1="150" x2="700" y2="250"></line>
                <line x1="700" y1="250" x2="900" y2="200"></line>
                <line x1="900" y1="200" x2="1100" y2="300"></line>
                <line x1="100" y1="300" x2="350" y2="400"></line>
                <line x1="350" y1="400" x2="600" y2="350"></line>
                <line x1="600" y1="350" x2="850" y2="450"></line>
                <line x1="850" y1="450" x2="1100" y2="400"></line>
                <line x1="200" y1="500" x2="400" y2="600"></line>
                <line x1="400" y1="600" x2="650" y2="550"></line>
                <line x1="650" y1="550" x2="900" y2="650"></line>
                <line x1="100" y1="100" x2="200" y2="500"></line>
                <line x1="300" y1="200" x2="350" y2="400"></line>
                <line x1="500" y1="150" x2="600" y2="350"></line>
                <line x1="700" y1="250" x2="650" y2="550"></line>
                <line x1="900" y1="200" x2="850" y2="450"></line>
            </g>
            <g fill="url(#cell-node-gradient)">
                <circle cx="100" cy="100" r="5"></circle>
                <circle cx="300" cy="200" r="5"></circle>
                <circle cx="500" cy="150" r="5"></circle>
                <circle cx="700" cy="250" r="5"></circle>
                <circle cx="900" cy="200" r="5"></circle>
                <circle cx="1100" cy="300" r="5"></circle>
                <circle cx="100" cy="300" r="5"></circle>
                <circle cx="350" cy="400" r="5"></circle>
                <circle cx="600" cy="350" r="5"></circle>
                <circle cx="850" cy="450" r="5"></circle>
                <circle cx="1100" cy="400" r="5"></circle>
                <circle cx="200" cy="500" r="5"></circle>
                <circle cx="400" cy="600" r="5"></circle>
                <circle cx="650" cy="550" r="5"></circle>
                <circle cx="900" cy="650" r="5"></circle>
            </g>
            <defs>
                <linearGradient id="cell-gradient" x1="0" x2="1200" y1="0" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#3B82F6"></stop>
                    <stop offset="1" stopColor="#8B5CF6"></stop>
                </linearGradient>
                <linearGradient id="cell-node-gradient" x1="0" x2="1200" y1="0" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#6366F1"></stop>
                    <stop offset="1" stopColor="#EC4899"></stop>
                </linearGradient>
            </defs>
        </svg>
    );
}

// 18. Gradient Orbs Background
export function SvgGradientOrbsBackground({ orbCount = 5 }: { orbCount?: number }) {
    const orbs = Array.from({ length: orbCount }, (_, i) => ({
        cx: Math.random() * 1200,
        cy: Math.random() * 800,
        r: Math.random() * 200 + 100,
        gradientId: `orb-${i}`
    }));

    const colors = [
        ['#FF6B9D', '#C44569'],
        ['#4FACFE', '#00F2FE'],
        ['#43E97B', '#38F9D7'],
        ['#FA709A', '#FEE140'],
        ['#A8E6CF', '#3D84A8']
    ];

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <g filter="url(#orb-blur)">
                {orbs.map((orb, i) => (
                    <circle key={i} cx={orb.cx} cy={orb.cy} r={orb.r} fill={`url(#${orb.gradientId})`} opacity="0.6"></circle>
                ))}
            </g>
            <defs>
                {orbs.map((orb, i) => (
                    <radialGradient key={i} id={orb.gradientId} cx={orb.cx} cy={orb.cy} r={orb.r} gradientUnits="userSpaceOnUse">
                        <stop stopColor={colors[i % colors.length][0]}></stop>
                        <stop offset="1" stopColor={colors[i % colors.length][1]} stopOpacity="0"></stop>
                    </radialGradient>
                ))}
                <filter id="orb-blur">
                    <feGaussianBlur stdDeviation="50"></feGaussianBlur>
                </filter>
            </defs>
        </svg>
    );
}

// 19. Squares Grid Background
export function SvgSquaresGridBackground({ squareSize = 60, gap = 10, colors = ['#6366F1', '#8B5CF6', '#EC4899'] }: { squareSize?: number; gap?: number; colors?: string[] }) {
    const cols = Math.ceil(1200 / (squareSize + gap));
    const rows = Math.ceil(800 / (squareSize + gap));
    const squares = [];

    for (let row = 0; row < rows; row++) {
        for (let col = 0; col < cols; col++) {
            squares.push({
                x: col * (squareSize + gap),
                y: row * (squareSize + gap),
                color: colors[Math.floor(Math.random() * colors.length)],
                opacity: Math.random() * 0.5 + 0.2
            });
        }
    }

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            {squares.map((square, i) => (
                <rect
                    key={i}
                    x={square.x}
                    y={square.y}
                    width={squareSize}
                    height={squareSize}
                    rx="8"
                    fill={square.color}
                    opacity={square.opacity}
                ></rect>
            ))}
        </svg>
    );
}

// 20. Topographic Lines Background
export function SvgTopographicBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full" opacity="0.3">
            <g stroke="#6366F1" strokeWidth="1.5" fill="none">
                <ellipse cx="600" cy="400" rx="100" ry="80"></ellipse>
                <ellipse cx="600" cy="400" rx="150" ry="120"></ellipse>
                <ellipse cx="600" cy="400" rx="200" ry="160"></ellipse>
                <ellipse cx="600" cy="400" rx="250" ry="200"></ellipse>
                <ellipse cx="600" cy="400" rx="300" ry="240"></ellipse>
                <ellipse cx="600" cy="400" rx="350" ry="280"></ellipse>
                <ellipse cx="600" cy="400" rx="400" ry="320"></ellipse>
                <ellipse cx="200" cy="200" rx="80" ry="60"></ellipse>
                <ellipse cx="200" cy="200" rx="120" ry="90"></ellipse>
                <ellipse cx="200" cy="200" rx="160" ry="120"></ellipse>
                <ellipse cx="1000" cy="600" rx="100" ry="70"></ellipse>
                <ellipse cx="1000" cy="600" rx="150" ry="105"></ellipse>
                <ellipse cx="1000" cy="600" rx="200" ry="140"></ellipse>
            </g>
        </svg>
    );
}

// 21. Particle Field Background
export function SvgParticleFieldBackground({ particleCount = 100 }: { particleCount?: number }) {
    const particles = Array.from({ length: particleCount }, () => ({
        cx: Math.random() * 1200,
        cy: Math.random() * 800,
        r: Math.random() * 3 + 1,
        opacity: Math.random() * 0.8 + 0.2,
        color: ['#3B82F6', '#8B5CF6', '#EC4899', '#F59E0B'][Math.floor(Math.random() * 4)]
    }));

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            {particles.map((particle, i) => (
                <circle
                    key={i}
                    cx={particle.cx}
                    cy={particle.cy}
                    r={particle.r}
                    fill={particle.color}
                    opacity={particle.opacity}
                ></circle>
            ))}
        </svg>
    );
}

// 22. Mountain Silhouette Background
export function SvgMountainBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <path d="M0,800 L0,500 L200,300 L400,450 L600,200 L800,400 L1000,250 L1200,450 L1200,800 Z" fill="url(#mountain-a)" opacity="0.7"></path>
            <path d="M0,800 L0,600 L150,450 L350,550 L550,350 L750,500 L950,400 L1200,550 L1200,800 Z" fill="url(#mountain-b)" opacity="0.6"></path>
            <path d="M0,800 L0,650 L200,550 L400,600 L600,500 L800,580 L1000,520 L1200,620 L1200,800 Z" fill="url(#mountain-c)" opacity="0.5"></path>
            <defs>
                <linearGradient id="mountain-a" x1="600" x2="600" y1="200" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#4F46E5"></stop>
                    <stop offset="1" stopColor="#7C3AED"></stop>
                </linearGradient>
                <linearGradient id="mountain-b" x1="600" x2="600" y1="350" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#7C3AED"></stop>
                    <stop offset="1" stopColor="#A78BFA"></stop>
                </linearGradient>
                <linearGradient id="mountain-c" x1="600" x2="600" y1="500" y2="800" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#A78BFA"></stop>
                    <stop offset="1" stopColor="#C4B5FD"></stop>
                </linearGradient>
            </defs>
        </svg>
    );
}

// 23. Spiral Pattern Background
export function SvgSpiralBackground({ spirals = 3, colors = ['#FF6B9D', '#4FACFE', '#43E97B'] }: { spirals?: number; colors?: string[] }) {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full" opacity="0.4">
            <g transform="translate(600, 400)">
                {Array.from({ length: spirals }, (_, i) => {
                    const rotation = (360 / spirals) * i;
                    const color = colors[i % colors.length];
                    return (
                        <g key={i} transform={`rotate(${rotation})`}>
                            <path
                                d="M0,0 Q100,-100 200,0 T400,100 Q450,200 400,300 T200,400 Q0,450 -200,400 T-400,100 Q-450,-200 -200,-300"
                                stroke={color}
                                strokeWidth="3"
                                fill="none"
                            ></path>
                        </g>
                    );
                })}
            </g>
        </svg>
    );
}

// 24. Bokeh Circles Background
export function SvgBokehBackground({ circleCount = 30 }: { circleCount?: number }) {
    const circles = Array.from({ length: circleCount }, () => ({
        cx: Math.random() * 1200,
        cy: Math.random() * 800,
        r: Math.random() * 80 + 20,
        opacity: Math.random() * 0.3 + 0.1,
        color: ['#FF6B9D', '#4FACFE', '#43E97B', '#FEE140', '#A78BFA'][Math.floor(Math.random() * 5)]
    }));

    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <g filter="url(#bokeh-blur)">
                {circles.map((circle, i) => (
                    <circle
                        key={i}
                        cx={circle.cx}
                        cy={circle.cy}
                        r={circle.r}
                        fill={circle.color}
                        opacity={circle.opacity}
                    ></circle>
                ))}
            </g>
            <defs>
                <filter id="bokeh-blur">
                    <feGaussianBlur stdDeviation="25"></feGaussianBlur>
                </filter>
            </defs>
        </svg>
    );
}

// 25. Liquid Morphing Background
export function SvgLiquidBackground() {
    return (
        <svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" fill="none" className="absolute inset-0 -z-10 w-full h-full">
            <g filter="url(#liquid-goo)">
                <path
                    d="M300,200 Q400,100 500,200 T700,300 Q750,400 650,500 T450,600 Q300,650 200,550 T200,350 Q200,250 300,200"
                    fill="url(#liquid-a)"
                    opacity="0.8"
                ></path>
                <path
                    d="M700,150 Q850,100 950,200 T1100,350 Q1150,500 1000,600 T750,650 Q650,600 600,500 T650,300 Q700,200 700,150"
                    fill="url(#liquid-b)"
                    opacity="0.7"
                ></path>
                <path
                    d="M150,500 Q250,450 350,500 T550,600 Q600,700 500,750 T350,700 Q250,650 200,550 T150,500"
                    fill="url(#liquid-c)"
                    opacity="0.6"
                ></path>
            </g>
            <defs>
                <linearGradient id="liquid-a" x1="200" x2="700" y1="200" y2="600" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#667EEA"></stop>
                    <stop offset="1" stopColor="#764BA2"></stop>
                </linearGradient>
                <linearGradient id="liquid-b" x1="600" x2="1150" y1="150" y2="650" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#F093FB"></stop>
                    <stop offset="1" stopColor="#F5576C"></stop>
                </linearGradient>
                <linearGradient id="liquid-c" x1="150" x2="600" y1="500" y2="750" gradientUnits="userSpaceOnUse">
                    <stop stopColor="#4FACFE"></stop>
                    <stop offset="1" stopColor="#00F2FE"></stop>
                </linearGradient>
                <filter id="liquid-goo" colorInterpolationFilters="sRGB">
                    <feGaussianBlur in="SourceGraphic" stdDeviation="15" result="blur"></feGaussianBlur>
                    <feColorMatrix in="blur" mode="matrix" values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 18 -7" result="goo"></feColorMatrix>
                    <feComposite in="SourceGraphic" in2="goo" operator="atop"></feComposite>
                </filter>
            </defs>
        </svg>
    );
}

export default {
    SvgGradientBackground,
    SvgWaveBackground,
    SvgCircleBurstBackground,
    SvgPolygonBackground,
    SvgMeshBackground,
    SvgStripedBackground,
    SvgHexagonBackground,
    SvgBlobBackground,
    SvgGridDotsBackground,
    SvgTriangleMosaicBackground,
    SvgCurvedLinesBackground,
    SvgStarFieldBackground,
    SvgAbstractFlowBackground,
    SvgRadialBurstBackground,
    SvgLayeredShapesBackground,
    SvgDiagonalLinesBackground,
    SvgCellularBackground,
    SvgGradientOrbsBackground,
    SvgSquaresGridBackground,
    SvgTopographicBackground,
    SvgParticleFieldBackground,
    SvgMountainBackground,
    SvgSpiralBackground,
    SvgBokehBackground,
    SvgLiquidBackground
};
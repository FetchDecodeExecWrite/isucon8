/*                                                                -*- C -*-
   +----------------------------------------------------------------------+
   | PHP Version 7                                                        |
   +----------------------------------------------------------------------+
   | Copyright (c) 1997-2018 The PHP Group                                |
   +----------------------------------------------------------------------+
   | This source file is subject to version 3.01 of the PHP license,      |
   | that is bundled with this package in the file LICENSE, and is        |
   | available through the world-wide-web at the following url:           |
   | http://www.php.net/license/3_01.txt                                  |
   | If you did not receive a copy of the PHP license and are unable to   |
   | obtain it through the world-wide-web, please send a note to          |
   | license@php.net so we can mail you a copy immediately.               |
   +----------------------------------------------------------------------+
   | Author: Stig Sæther Bakken <ssb@php.net>                             |
   +----------------------------------------------------------------------+
*/

/* $Id$ */

#define CONFIGURE_COMMAND " './configure'  '--with-config-file-path=/home/isucon/local/php/etc' '--with-config-file-scan-dir=/home/isucon/local/php/etc/conf.d' '--prefix=/home/isucon/local/php' '--libexecdir=/home/isucon/local/php/libexec' '--without-pear' '--with-gd' '--with-jpeg-dir=/usr' '--with-png-dir=/usr' '--enable-exif' '--with-zlib-dir=/usr' '--enable-intl' '--with-kerberos' '--with-mcrypt=/usr' '--enable-soap' '--enable-xmlreader' '--with-xsl' '--enable-ftp' '--enable-cgi' '--with-curl=/usr' '--with-tidy' '--with-xmlrpc' '--with-pdo-sqlite' '--with-readline' '--disable-debug' '--enable-phpdbg' '--with-zlib' '--enable-fpm' '--enable-pdo' '--with-pear' '--with-mysqli=mysqlnd' '--with-pdo-mysql=mysqlnd' '--with-openssl' '--with-pcre-regex' '--with-pcre-dir' '--with-libxml-dir' '--enable-opcache' '--enable-bcmath' '--with-bz2' '--enable-calendar' '--enable-cli' '--enable-shmop' '--enable-sysvsem' '--enable-sysvshm' '--enable-sysvmsg' '--enable-mbregex' '--enable-mbstring' '--enable-pcntl' '--enable-sockets' '--with-curl' '--enable-zip' '--with-libdir=lib64'"
#define PHP_ADA_INCLUDE		""
#define PHP_ADA_LFLAGS		""
#define PHP_ADA_LIBS		""
#define PHP_APACHE_INCLUDE	""
#define PHP_APACHE_TARGET	""
#define PHP_FHTTPD_INCLUDE      ""
#define PHP_FHTTPD_LIB          ""
#define PHP_FHTTPD_TARGET       ""
#define PHP_CFLAGS		"$(CFLAGS_CLEAN) -prefer-non-pic -static"
#define PHP_DBASE_LIB		""
#define PHP_BUILD_DEBUG		""
#define PHP_GDBM_INCLUDE	""
#define PHP_IBASE_INCLUDE	""
#define PHP_IBASE_LFLAGS	""
#define PHP_IBASE_LIBS		""
#define PHP_IFX_INCLUDE		""
#define PHP_IFX_LFLAGS		""
#define PHP_IFX_LIBS		""
#define PHP_INSTALL_IT		""
#define PHP_IODBC_INCLUDE	""
#define PHP_IODBC_LFLAGS	""
#define PHP_IODBC_LIBS		""
#define PHP_MSQL_INCLUDE	""
#define PHP_MSQL_LFLAGS		""
#define PHP_MSQL_LIBS		""
#define PHP_MYSQL_INCLUDE	"@MYSQL_INCLUDE@"
#define PHP_MYSQL_LIBS		"@MYSQL_LIBS@"
#define PHP_MYSQL_TYPE		"@MYSQL_MODULE_TYPE@"
#define PHP_ODBC_INCLUDE	""
#define PHP_ODBC_LFLAGS		""
#define PHP_ODBC_LIBS		""
#define PHP_ODBC_TYPE		""
#define PHP_OCI8_SHARED_LIBADD 	""
#define PHP_OCI8_DIR			""
#define PHP_OCI8_ORACLE_VERSION		""
#define PHP_ORACLE_SHARED_LIBADD 	"@ORACLE_SHARED_LIBADD@"
#define PHP_ORACLE_DIR				"@ORACLE_DIR@"
#define PHP_ORACLE_VERSION			"@ORACLE_VERSION@"
#define PHP_PGSQL_INCLUDE	""
#define PHP_PGSQL_LFLAGS	""
#define PHP_PGSQL_LIBS		""
#define PHP_PROG_SENDMAIL	"/sbin/sendmail"
#define PHP_SOLID_INCLUDE	""
#define PHP_SOLID_LIBS		""
#define PHP_EMPRESS_INCLUDE	""
#define PHP_EMPRESS_LIBS	""
#define PHP_SYBASE_INCLUDE	""
#define PHP_SYBASE_LFLAGS	""
#define PHP_SYBASE_LIBS		""
#define PHP_DBM_TYPE		""
#define PHP_DBM_LIB		""
#define PHP_LDAP_LFLAGS		""
#define PHP_LDAP_INCLUDE	""
#define PHP_LDAP_LIBS		""
#define PHP_BIRDSTEP_INCLUDE     ""
#define PHP_BIRDSTEP_LIBS        ""
#define PEAR_INSTALLDIR         "/home/isucon/local/php/lib/php"
#define PHP_INCLUDE_PATH	".:/home/isucon/local/php/lib/php"
#define PHP_EXTENSION_DIR       "/home/isucon/local/php/lib/php/extensions/no-debug-non-zts-20170718"
#define PHP_PREFIX              "/home/isucon/local/php"
#define PHP_BINDIR              "/home/isucon/local/php/bin"
#define PHP_SBINDIR             "/home/isucon/local/php/sbin"
#define PHP_MANDIR              "/home/isucon/local/php/php/man"
#define PHP_LIBDIR              "/home/isucon/local/php/lib/php"
#define PHP_DATADIR             "/home/isucon/local/php/share/php"
#define PHP_SYSCONFDIR          "/home/isucon/local/php/etc"
#define PHP_LOCALSTATEDIR       "/home/isucon/local/php/var"
#define PHP_CONFIG_FILE_PATH    "/home/isucon/local/php/etc"
#define PHP_CONFIG_FILE_SCAN_DIR    "/home/isucon/local/php/etc/conf.d"
#define PHP_SHLIB_SUFFIX        "so"
#define PHP_SHLIB_EXT_PREFIX    ""
